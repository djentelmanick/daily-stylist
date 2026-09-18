import { useEffect, useRef, useState } from 'react'
import { useBackButton } from './telegram'
import { texts } from './texts'

type Facing = 'environment' | 'user'

export type CameraState =
  | { kind: 'closed' }
  | { kind: 'starting' }
  | { kind: 'open'; stream: MediaStream; facing: Facing }
  | { kind: 'failed'; message: string }

const PORTRAIT = 3 / 4

// Telegram спрашивает разрешение на каждое включение, поэтому камера живёт минуту после съёмки
// и одна на всё приложение: форма пересоздаётся после каждой добавленной вещи.
const IDLE_TIMEOUT_MS = 60_000

const lensKey = 'camera:lens'
const zoomKey = 'camera:zoom'

let facing: Facing = 'environment'
let shared: MediaStream | null = null
let pending: Promise<MediaStream> | null = null
let idleTimer: number | undefined

export function cameraSupported(): boolean {
  return typeof navigator.mediaDevices?.getUserMedia === 'function'
}

async function acquire(): Promise<MediaStream> {
  window.clearTimeout(idleTimer)
  if (shared !== null && live(shared) && matches(shared)) {
    return shared
  }
  if (shared !== null) {
    stop(shared)
    shared = null
  }
  pending ??= request().finally(() => {
    pending = null
  })
  shared = await pending
  return shared
}

async function request(): Promise<MediaStream> {
  try {
    return await navigator.mediaDevices.getUserMedia({ video: constraints(), audio: false })
  } catch (error) {
    // Запомненной камеры больше нет - берём заднюю по умолчанию.
    if (facing === 'environment' && load(lensKey) !== null && isError(error, 'OverconstrainedError')) {
      save(lensKey, null)
      return navigator.mediaDevices.getUserMedia({ video: constraints(), audio: false })
    }
    throw error
  }
}

function constraints(): MediaTrackConstraints {
  const size = { aspectRatio: { ideal: 4 / 3 }, width: { ideal: 1920 }, height: { ideal: 1440 } }
  if (facing === 'user') {
    return { ...size, facingMode: { ideal: 'user' } }
  }
  const lens = load(lensKey)
  return lens === null ? { ...size, facingMode: { ideal: 'environment' } } : { ...size, deviceId: { exact: lens } }
}

function matches(stream: MediaStream): boolean {
  const settings = stream.getVideoTracks()[0]?.getSettings()
  if (settings === undefined) {
    return false
  }
  if (facing === 'user') {
    return settings.facingMode === 'user'
  }
  const lens = load(lensKey)
  return lens === null ? settings.facingMode !== 'user' : settings.deviceId === lens
}

function releaseLater() {
  window.clearTimeout(idleTimer)
  idleTimer = window.setTimeout(() => {
    if (shared !== null) {
      stop(shared)
      shared = null
    }
  }, IDLE_TIMEOUT_MS)
}

// Камера включается по нажатию, а не в эффекте: эффект в режиме разработки
// срабатывает дважды, и телефон спрашивал разрешение два раза.
export function useCamera() {
  const [state, setState] = useState<CameraState>({ kind: 'closed' })
  const attempt = useRef(0)
  const holding = useRef(false)

  useEffect(
    () => () => {
      attempt.current++
      if (holding.current) {
        holding.current = false
        releaseLater()
      }
    },
    [],
  )

  async function open() {
    const current = ++attempt.current
    setState({ kind: 'starting' })
    try {
      const stream = await acquire()
      holding.current = true
      // Камеру успели закрыть, пока телефон спрашивал разрешение.
      if (attempt.current !== current) {
        holding.current = false
        releaseLater()
        return
      }
      setState({ kind: 'open', stream, facing })
    } catch (error) {
      if (attempt.current === current) {
        setState({ kind: 'failed', message: cameraError(error) })
      }
    }
  }

  function flip() {
    facing = facing === 'user' ? 'environment' : 'user'
    return open()
  }

  function chooseLens(lens: string) {
    facing = 'environment'
    save(lensKey, lens)
    save(zoomKey, null)
    return open()
  }

  function close() {
    attempt.current++
    if (holding.current) {
      holding.current = false
      releaseLater()
    }
    setState({ kind: 'closed' })
  }

  return { state, open, close, flip, chooseLens }
}

export function Camera({
  state,
  onCapture,
  onFlip,
  onChooseLens,
  onClose,
}: {
  state: CameraState
  onCapture: (photo: File) => void
  onFlip: () => void
  onChooseLens: (lens: string) => void
  onClose: () => void
}) {
  useBackButton(onClose)
  const video = useRef<HTMLVideoElement>(null)
  const stream = state.kind === 'open' ? state.stream : null
  const [lenses, setLenses] = useState<string[]>([])
  const [hasFront, setHasFront] = useState(false)
  const [zooms, setZooms] = useState<number[]>([])
  const [zoom, setZoom] = useState(0)

  useEffect(() => {
    if (video.current !== null) {
      video.current.srcObject = stream
    }
  }, [stream])

  async function prepare() {
    const track = stream?.getVideoTracks()[0]
    if (track === undefined) {
      return
    }
    const cameras = (await navigator.mediaDevices.enumerateDevices()).filter((device) => device.kind === 'videoinput')
    const back = cameras.filter((camera) => facingOf(camera) !== 'user')
    setLenses(back.map((camera) => camera.deviceId))
    setHasFront(back.length < cameras.length)

    // Зум нужен там, где задняя камера одна: несколько объективов выбираются сами по себе.
    const range = zoomRange(track)
    const steps = range === null || back.length > 1 ? [] : zoomSteps(range.min, range.max)
    if (steps.length < 2) {
      setZooms([])
      return
    }
    const preferred = Number(load(zoomKey) ?? 1)
    const chosen = steps.includes(preferred) ? preferred : steps.includes(1) ? 1 : steps[0]
    try {
      await applyZoom(track, chosen)
    } catch {
      return
    }
    setZooms(steps)
    setZoom(chosen)
  }

  async function chooseZoom(value: number) {
    const track = stream?.getVideoTracks()[0]
    if (track === undefined) {
      return
    }
    try {
      await applyZoom(track, value)
    } catch {
      return
    }
    setZoom(value)
    save(zoomKey, String(value))
  }

  function shoot() {
    const element = video.current
    if (element === null || element.videoWidth === 0) {
      return
    }
    // До 3:4, как и превью: телефон может отдать и 16:9, и квадрат.
    const crop = centerCrop(element.videoWidth, element.videoHeight)
    const canvas = document.createElement('canvas')
    canvas.width = crop.width
    canvas.height = crop.height
    canvas.getContext('2d')?.drawImage(element, crop.x, crop.y, crop.width, crop.height, 0, 0, crop.width, crop.height)
    canvas.toBlob(
      (blob) => {
        if (blob !== null) {
          onCapture(new File([blob], 'camera.jpg', { type: 'image/jpeg' }))
        }
      },
      'image/jpeg',
      0.92,
    )
  }

  const open = state.kind === 'open'
  const front = open && state.facing === 'user'
  const currentLens = stream?.getVideoTracks()[0]?.getSettings().deviceId

  return (
    <div className="camera">
      <div className="camera-frame">
        {state.kind === 'failed' ? (
          <p className="camera-message">{state.message}</p>
        ) : (
          <video
            ref={video}
            className={front ? 'camera-view camera-view-mirrored' : 'camera-view'}
            autoPlay
            playsInline
            muted
            onLoadedMetadata={() => void prepare()}
          />
        )}
      </div>
      {open && !front && lenses.length > 1 && (
        <div className="camera-choices">
          {lenses.map((lens, index) => (
            <button
              key={lens}
              type="button"
              className={lens === currentLens ? 'camera-choice camera-choice-active' : 'camera-choice'}
              aria-label={texts.lens(index + 1)}
              onClick={() => onChooseLens(lens)}
            >
              {index + 1}
            </button>
          ))}
        </div>
      )}
      {open && zooms.length > 1 && (
        <div className="camera-choices">
          {zooms.map((value) => (
            <button
              key={value}
              type="button"
              className={value === zoom ? 'camera-choice camera-choice-active' : 'camera-choice'}
              onClick={() => void chooseZoom(value)}
            >
              {zoomLabel(value)}
            </button>
          ))}
        </div>
      )}
      <div className="camera-controls">
        <button type="button" className="camera-cancel" onClick={onClose}>
          {texts.cancel}
        </button>
        <button type="button" className="camera-shutter" aria-label={texts.shoot} disabled={!open} onClick={shoot} />
        {open && hasFront ? (
          <button type="button" className="camera-flip" aria-label={texts.flipCamera} onClick={onFlip}>
            <FlipIcon />
          </button>
        ) : (
          <span />
        )}
      </div>
    </div>
  )
}

function FlipIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path
        fill="currentColor"
        d="M12 5V2L8 6l4 4V7a5 5 0 0 1 4.9 6h2.05A7 7 0 0 0 12 5Zm0 12a5 5 0 0 1-4.9-6H5.05A7 7 0 0 0 12 19v3l4-4-4-4v3Z"
      />
    </svg>
  )
}

function facingOf(camera: MediaDeviceInfo): Facing | undefined {
  const reported = (camera as InputDeviceInfo).getCapabilities?.().facingMode
  if (reported !== undefined && reported.length > 0) {
    return reported.includes('user') ? 'user' : 'environment'
  }
  return /front/i.test(camera.label) ? 'user' : undefined
}

function zoomRange(track: MediaStreamTrack): { min: number; max: number } | null {
  const capabilities = (track.getCapabilities?.() ?? {}) as { zoom?: { min: number; max: number } }
  return capabilities.zoom ?? null
}

function zoomSteps(min: number, max: number): number[] {
  return [...new Set([min, 1, 2].filter((value) => value >= min && value <= max))].sort((a, b) => a - b)
}

function zoomLabel(value: number): string {
  return `${Number(value.toFixed(1))}x`
}

function applyZoom(track: MediaStreamTrack, value: number): Promise<void> {
  return track.applyConstraints({ advanced: [{ zoom: value } as MediaTrackConstraintSet] })
}

function live(stream: MediaStream): boolean {
  return stream.getVideoTracks().some((track) => track.readyState === 'live')
}

function centerCrop(width: number, height: number) {
  if (width / height > PORTRAIT) {
    const cropWidth = Math.round(height * PORTRAIT)
    return { x: Math.round((width - cropWidth) / 2), y: 0, width: cropWidth, height }
  }
  const cropHeight = Math.round(width / PORTRAIT)
  return { x: 0, y: Math.round((height - cropHeight) / 2), width, height: cropHeight }
}

function stop(stream: MediaStream) {
  for (const track of stream.getTracks()) {
    track.stop()
  }
}

// Выбор объектива нужен и в следующий раз, а не только до перезагрузки страницы.
function load(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

function save(key: string, value: string | null) {
  try {
    if (value === null) {
      localStorage.removeItem(key)
    } else {
      localStorage.setItem(key, value)
    }
  } catch {
    // Хранилище запрещено - выбор просто не запомнится.
  }
}

function isError(error: unknown, name: string): boolean {
  return error instanceof DOMException && error.name === name
}

function cameraError(error: unknown): string {
  if (isError(error, 'NotAllowedError') || isError(error, 'SecurityError')) {
    return texts.cameraDenied
  }
  if (isError(error, 'NotFoundError') || isError(error, 'OverconstrainedError')) {
    return texts.cameraMissing
  }
  return texts.cameraFailed
}

import { useEffect, useRef } from 'react'

type TelegramWebApp = {
  initData: string
  colorScheme?: 'light' | 'dark'
  ready: () => void
  expand: () => void
  isVersionAtLeast: (version: string) => boolean
  showConfirm: (message: string, callback: (confirmed: boolean) => void) => void
  BackButton: {
    show: () => void
    hide: () => void
    onClick: (callback: () => void) => void
    offClick: (callback: () => void) => void
  }
  HapticFeedback: {
    impactOccurred: (style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft') => void
  }
  LocationManager: {
    isInited: boolean
    isLocationAvailable: boolean
    isAccessRequested: boolean
    isAccessGranted: boolean
    init: (callback: () => void) => void
    getLocation: (callback: (location: Position | null) => void) => void
    openSettings: () => void
  }
}

declare global {
  interface Window {
    Telegram?: { WebApp?: TelegramWebApp }
  }
}

const webApp = window.Telegram?.WebApp

export function initTelegram(): void {
  webApp?.ready()
  webApp?.expand()
  // Без color-scheme нативные поля вроде выбора времени в тёмной теме остаются светлыми.
  if (webApp?.colorScheme !== undefined) {
    document.documentElement.style.colorScheme = webApp.colorScheme
  }
}

export function getInitData(): string {
  return webApp?.initData ?? ''
}

// Нажатие получает только верхний обработчик: экран под открытой камерой не должен
// закрыться вместе с ней.
const backHandlers: { current: () => void }[] = []

function handleBack() {
  backHandlers[backHandlers.length - 1]?.current()
}

export function useBackButton(onBack: () => void): void {
  const latest = useRef(onBack)
  useEffect(() => {
    latest.current = onBack
  })

  useEffect(() => {
    if (!webApp?.isVersionAtLeast('6.1')) {
      return
    }
    const button = webApp.BackButton
    if (backHandlers.length === 0) {
      button.onClick(handleBack)
      button.show()
    }
    backHandlers.push(latest)
    return () => {
      backHandlers.splice(backHandlers.indexOf(latest), 1)
      if (backHandlers.length === 0) {
        button.offClick(handleBack)
        button.hide()
      }
    }
  }, [])
}

// Нативный диалог Telegram: window.confirm в части клиентов не работает.
export function confirm(message: string): Promise<boolean> {
  const app = webApp
  if (!app?.isVersionAtLeast('6.2')) {
    return Promise.resolve(window.confirm(message))
  }
  return new Promise((resolve) => app.showConfirm(message, resolve))
}

// Не light и не selectionChanged: на Android их вибрацию не почувствовать.
export function vibrate(): void {
  if (webApp?.isVersionAtLeast('6.1')) {
    webApp.HapticFeedback.impactOccurred('medium')
  }
}

export type Position = {
  latitude: number
  longitude: number
}

export class PositionError extends Error {
  readonly denied: boolean

  constructor(denied: boolean) {
    super(denied ? 'denied' : 'unavailable')
    this.denied = denied
  }
}

// Сначала геопозиция Telegram: браузерная внутри Telegram на части клиентов не работает.
export function getPosition(): Promise<Position> {
  const manager = webApp?.isVersionAtLeast('8.0') ? webApp.LocationManager : undefined
  if (manager === undefined) {
    return browserPosition()
  }
  return new Promise((resolve, reject) => {
    const request = () => {
      if (!manager.isLocationAvailable) {
        browserPosition().then(resolve, reject)
        return
      }
      manager.getLocation((location) => {
        if (location === null) {
          reject(new PositionError(true))
        } else {
          resolve({ latitude: location.latitude, longitude: location.longitude })
        }
      })
    }
    if (manager.isInited) {
      request()
    } else {
      manager.init(request)
    }
  })
}

// Второй раз Telegram не спрашивает: доступ, в котором отказали, включается только в настройках.
export function canOpenLocationSettings(): boolean {
  const manager = webApp?.isVersionAtLeast('8.0') ? webApp.LocationManager : undefined
  return manager !== undefined && manager.isAccessRequested && !manager.isAccessGranted
}

export function openLocationSettings(): void {
  webApp?.LocationManager.openSettings()
}

const positionTimeoutMs = 15_000
const positionMaxAgeMs = 10 * 60 * 1000

function browserPosition(): Promise<Position> {
  return new Promise((resolve, reject) => {
    if (!('geolocation' in navigator)) {
      reject(new PositionError(false))
      return
    }
    navigator.geolocation.getCurrentPosition(
      (position) => resolve({ latitude: position.coords.latitude, longitude: position.coords.longitude }),
      (error) => reject(new PositionError(error.code === error.PERMISSION_DENIED)),
      { timeout: positionTimeoutMs, maximumAge: positionMaxAgeMs },
    )
  })
}

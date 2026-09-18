import { useEffect, useId, useLayoutEffect, useRef, useState, type ChangeEvent, type FormEvent } from 'react'
import {
  recognizePhoto,
  uploadPhoto,
  type Item,
  type ItemFields,
  type Option,
  type Options,
  type Suggestion,
  type WarmthLevelOption,
} from './api'
import { Camera, cameraSupported, useCamera } from './Camera'
import { preparePhoto } from './photo'
import { forget, recall, remember } from './session'
import { errorText, errorTexts, texts } from './texts'
import { Chip, Dots, Swatch, Thinking } from './ui'

type Status = { kind: 'idle' } | { kind: 'submitting' } | { kind: 'failed'; message: string }

type Props = {
  options: Options
  title: string
  submitText: string
  draftKey: string
  initial?: Item
  notice?: string
  onOpenPhoto: (url: string, caption: string) => void
  onSubmit: (fields: ItemFields) => Promise<void>
}

// Что вписала модель: уходит вместе с фотографией, не задевая правок человека.
type Suggested = Partial<Suggestion>

type Draft = ItemFields & {
  photo_url: string
  warmth_chosen: boolean
  suggested: Suggested
}

export function forgetDraft(draftKey: string): void {
  forget(draftKey)
}

export function ItemForm({ options, title, submitText, draftKey, initial, notice, onOpenPhoto, onSubmit }: Props) {
  const [draft] = useState(() => recall<Draft>(draftKey) ?? initial ?? null)
  const minWarmth = options.warmth_levels[0].value

  const [name, setName] = useState(draft?.name ?? '')
  const [description, setDescription] = useState(draft?.description ?? '')
  const [category, setCategory] = useState(draft?.category ?? '')
  const [mainColor, setMainColor] = useState(draft?.main_color ?? '')
  const [extraColors, setExtraColors] = useState<string[]>(draft?.extra_colors ?? [])
  const [seasons, setSeasons] = useState<string[]>(draft?.seasons ?? [])
  const [warmthLevel, setWarmthLevel] = useState(draft?.warmth_level || minWarmth)
  // «Пусто» от «лёгкая» не отличить, поэтому помним, трогал ли ползунок человек:
  // иначе подсказка затрёт его выбор, а выбор молча оставит единицу на пуховике.
  const [warmthChosen, setWarmthChosen] = useState(
    draft !== null && 'warmth_chosen' in draft ? draft.warmth_chosen : initial !== undefined,
  )
  const [waterproof, setWaterproof] = useState(draft?.waterproof ?? false)
  const [photo, setPhoto] = useState<Photo>({ key: draft?.photo_key ?? '', preview: draft?.photo_url ?? '' })
  const [photoUploading, setPhotoUploading] = useState(false)
  const [photoError, setPhotoError] = useState('')
  const [recognizing, setRecognizing] = useState(false)
  const camera = useCamera()
  const [suggested, setSuggested] = useState<Suggested>(
    draft !== null && 'suggested' in draft ? draft.suggested : {},
  )
  const [status, setStatus] = useState<Status>({ kind: 'idle' })

  useEffect(() => {
    const draft: Draft = {
      name,
      description,
      category,
      main_color: mainColor,
      extra_colors: extraColors,
      seasons,
      warmth_level: warmthLevel,
      waterproof,
      photo_key: photo.key,
      photo_url: photo.preview,
      warmth_chosen: warmthChosen,
      suggested,
    }
    remember(draftKey, draft)
  }, [
    draftKey,
    name,
    description,
    category,
    mainColor,
    extraColors,
    seasons,
    warmthLevel,
    warmthChosen,
    waterproof,
    photo,
    suggested,
  ])

  // За пару секунд ожидания фотографию могли сменить или убрать: подсказка тогда
  // уже не про неё.
  const photoKey = useRef(photo.key)

  // Ответ модели и загрузка фото приходят после ожидания, а поля к этому времени могли
  // поменяться: сверяться нужно с тем, что в форме сейчас, а не в момент выбора фото.
  const snapshot = { name, category, mainColor, extraColors, seasons, warmthLevel, warmthChosen, waterproof, suggested }
  const latest = useRef(snapshot)
  useLayoutEffect(() => {
    latest.current = snapshot
  })

  const needsWarmth = options.categories.find((option) => option.value === category)?.has_warmth ?? false
  const allSeasonsSelected = seasons.length === options.seasons.length
  const canSubmit =
    name.trim() !== '' &&
    category !== '' &&
    mainColor !== '' &&
    seasons.length > 0 &&
    !photoUploading &&
    status.kind !== 'submitting'

  async function choosePhoto(file: File) {
    setPhotoError('')
    setPhotoUploading(true)
    try {
      const photoFile = await preparePhoto(file)
      if (!options.limits.photo_types.includes(photoFile.type)) {
        setPhotoError(errorTexts.photo_type_unsupported)
        return
      }
      if (photoFile.size > options.limits.photo_bytes) {
        setPhotoError(texts.photoTooBig(Math.floor(options.limits.photo_bytes / 1024 / 1024)))
        return
      }

      const uploaded = await uploadPhoto(photoFile)
      forgetSuggested()
      photoKey.current = uploaded.key
      setPhoto({ key: uploaded.key, preview: uploaded.url })
      void recognize(uploaded.key)
    } catch (uploadError) {
      setPhotoError(errorText(uploadError))
    } finally {
      setPhotoUploading(false)
    }
  }

  async function recognize(key: string) {
    setRecognizing(true)
    try {
      const suggestion = await recognizePhoto(key)
      if (photoKey.current === key) {
        fillEmptyFields(suggestion)
      }
    } catch {
      // Не распозналось - пользователь заполнит поля сам, говорить тут не о чем.
    } finally {
      if (photoKey.current === key) {
        setRecognizing(false)
      }
    }
  }

  function fillEmptyFields(suggestion: Suggestion) {
    if (suggestion.category === '') {
      return
    }
    const now = latest.current
    const filled: Suggested = {}

    if (now.name.trim() === '' && suggestion.name !== '') {
      setName(suggestion.name)
      filled.name = suggestion.name
    }
    if (now.category === '') {
      setCategory(suggestion.category)
      filled.category = suggestion.category
    }
    let main = now.mainColor
    if (main === '' && suggestion.main_color !== '') {
      main = suggestion.main_color
      setMainColor(main)
      filled.main_color = main
    }
    const extras = suggestion.extra_colors.filter((color) => color !== main)
    if (now.extraColors.length === 0 && extras.length > 0) {
      setExtraColors(extras)
      filled.extra_colors = extras
    }
    if (now.seasons.length === 0 && suggestion.seasons.length > 0) {
      setSeasons(suggestion.seasons)
      filled.seasons = suggestion.seasons
    }
    if (!now.warmthChosen && suggestion.warmth_level > 0) {
      setWarmthLevel(suggestion.warmth_level)
      filled.warmth_level = suggestion.warmth_level
    }
    if (!now.waterproof && suggestion.waterproof) {
      setWaterproof(true)
      filled.waterproof = true
    }
    setSuggested(filled)
  }

  // Поле, которое человек успел поправить, остаётся его: сбрасывается только нетронутое.
  function forgetSuggested() {
    const now = latest.current
    const was = now.suggested

    if (was.name !== undefined && now.name === was.name) {
      setName('')
    }
    if (was.category !== undefined && now.category === was.category) {
      setCategory('')
    }
    if (was.main_color !== undefined && now.mainColor === was.main_color) {
      setMainColor('')
    }
    if (was.extra_colors !== undefined && sameValues(now.extraColors, was.extra_colors)) {
      setExtraColors([])
    }
    if (was.seasons !== undefined && sameValues(now.seasons, was.seasons)) {
      setSeasons([])
    }
    if (was.warmth_level !== undefined && !now.warmthChosen && now.warmthLevel === was.warmth_level) {
      setWarmthLevel(minWarmth)
    }
    if (was.waterproof !== undefined && now.waterproof) {
      setWaterproof(false)
    }
    setSuggested({})
  }

  function removePhoto() {
    setPhotoError('')
    forgetSuggested()
    photoKey.current = ''
    setPhoto({ key: '', preview: '' })
  }

  function chooseCategory(value: string) {
    setCategory(value)
    const withWarmth = options.categories.find((option) => option.value === value)?.has_warmth ?? false
    if (withWarmth && warmthLevel === 0) {
      setWarmthLevel(minWarmth)
    }
  }

  function chooseWarmth(level: number) {
    setWarmthChosen(true)
    setWarmthLevel(level)
  }

  function chooseMainColor(color: string) {
    setMainColor(color)
    setExtraColors((current) => current.filter((extra) => extra !== color))
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    setStatus({ kind: 'submitting' })
    try {
      await onSubmit({
        name: name.trim(),
        description: description.trim(),
        category,
        main_color: mainColor,
        extra_colors: inOptionsOrder(options.colors, extraColors).filter((color) => color !== mainColor),
        seasons: inOptionsOrder(options.seasons, seasons),
        warmth_level: needsWarmth ? warmthLevel : 0,
        waterproof,
        photo_key: photo.key,
      })
      forget(draftKey)
      setStatus({ kind: 'idle' })
    } catch (error) {
      setStatus({ kind: 'failed', message: errorText(error) })
    }
  }

  return (
    <form className="screen" onSubmit={submit}>
      <h1>{title}</h1>

      <fieldset className="field field-photo">
        <legend className="field-title">{texts.photo}</legend>
        <PhotoField
          preview={photo.preview}
          uploading={photoUploading}
          recognizing={recognizing}
          accept={options.limits.photo_types.join(',')}
          onOpen={() => onOpenPhoto(photo.preview, name.trim())}
          onCamera={cameraSupported() ? () => void camera.open() : undefined}
          onChoose={choosePhoto}
          onRemove={removePhoto}
        />
        {photoError !== '' && <p className="notice notice-error">{photoError}</p>}
        {photoUploading && <Thinking>{texts.photoUploading}</Thinking>}
        {recognizing && <Thinking>{texts.photoRecognizing}</Thinking>}
        {Object.keys(suggested).length > 0 && !recognizing && (
          <p className="photo-status">{texts.photoRecognized}</p>
        )}
      </fieldset>

      <label className="field">
        <FieldHeader title={texts.name} length={name.length} limit={options.limits.name} />
        <input
          type="text"
          value={name}
          maxLength={options.limits.name}
          placeholder={texts.namePlaceholder}
          onChange={(event) => setName(event.target.value)}
        />
      </label>

      <label className="field">
        <FieldHeader title={texts.description} length={description.length} limit={options.limits.description} />
        <textarea
          rows={2}
          value={description}
          maxLength={options.limits.description}
          placeholder={texts.descriptionPlaceholder}
          onChange={(event) => setDescription(event.target.value)}
        />
      </label>

      <fieldset className="field">
        <legend className="field-title">{texts.category}</legend>
        <div className="chips">
          {options.categories.map((option) => (
            <Chip key={option.value} selected={category === option.value} onClick={() => chooseCategory(option.value)}>
              {option.label}
            </Chip>
          ))}
        </div>
      </fieldset>

      <fieldset className="field">
        <legend className="field-title">{texts.mainColor}</legend>
        <div className="chips">
          {options.colors.map((option) => (
            <Chip key={option.value} selected={mainColor === option.value} onClick={() => chooseMainColor(option.value)}>
              <Swatch color={option.value} />
              {option.label}
            </Chip>
          ))}
        </div>
      </fieldset>

      <fieldset className="field">
        <legend className="field-title">{texts.extraColors}</legend>
        <div className="chips">
          {options.colors
            .filter((option) => option.value !== mainColor)
            .map((option) => (
              <Chip
                key={option.value}
                selected={extraColors.includes(option.value)}
                onClick={() => setExtraColors((current) => toggle(current, option.value))}
              >
                <Swatch color={option.value} />
                {option.label}
              </Chip>
            ))}
        </div>
      </fieldset>

      <fieldset className="field">
        <legend className="field-title">{texts.seasons}</legend>
        <div className="chips">
          <Chip
            selected={allSeasonsSelected}
            onClick={() => setSeasons(allSeasonsSelected ? [] : options.seasons.map((option) => option.value))}
          >
            {texts.allSeasons}
          </Chip>
          {options.seasons.map((option) => (
            <Chip
              key={option.value}
              selected={seasons.includes(option.value)}
              onClick={() => setSeasons((current) => toggle(current, option.value))}
            >
              {option.label}
            </Chip>
          ))}
        </div>
      </fieldset>

      {needsWarmth && (
        <fieldset className="field">
          <legend className="field-title">{texts.warmthLevel}</legend>
          <WarmthSlider levels={options.warmth_levels} value={warmthLevel} onChange={chooseWarmth} />
        </fieldset>
      )}

      <label className="checkbox">
        <input type="checkbox" checked={waterproof} onChange={(event) => setWaterproof(event.target.checked)} />
        {texts.waterproof}
      </label>

      {notice && <p className="notice">{notice}</p>}
      {status.kind === 'failed' && <p className="notice notice-error">{status.message}</p>}

      <button type="submit" className="submit" disabled={!canSubmit}>
        {status.kind === 'submitting' ? (
          <>
            {texts.submitting}
            <Dots />
          </>
        ) : (
          submitText
        )}
      </button>
      {camera.state.kind !== 'closed' && (
        <Camera
          state={camera.state}
          onFlip={() => void camera.flip()}
          onChooseLens={(lens) => void camera.chooseLens(lens)}
          onClose={camera.close}
          onCapture={(captured) => {
            camera.close()
            void choosePhoto(captured)
          }}
        />
      )}
    </form>
  )
}

type Photo = {
  key: string
  preview: string
}

function PhotoField({
  preview,
  uploading,
  recognizing,
  accept,
  onOpen,
  onCamera,
  onChoose,
  onRemove,
}: {
  preview: string
  uploading: boolean
  recognizing: boolean
  accept: string
  onOpen: () => void
  onCamera?: () => void
  onChoose: (file: File) => void
  onRemove: () => void
}) {
  const inputId = useId()

  function choose(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (file) {
      onChoose(file)
    }
  }

  const busy = uploading || recognizing
  const slot = busyOrNot(busy)
  const sources = (
    <>
      {onCamera && (
        <button type="button" className="button" disabled={uploading} onClick={onCamera}>
          {texts.takePhoto}
        </button>
      )}
      <label htmlFor={inputId} className="button photo-button" aria-disabled={uploading}>
        {texts.pickPhoto}
      </label>
      <input id={inputId} type="file" accept={accept} hidden disabled={uploading} onChange={choose} />
    </>
  )

  if (preview === '') {
    return (
      <div className="photo-field">
        <span className={slot}>
          <label htmlFor={inputId} className="photo-circle photo-circle-empty photo-picker">
            +
          </label>
        </span>
        <div className="photo-actions">{sources}</div>
      </div>
    )
  }

  return (
    <div className="photo-field">
      <span className={slot}>
        <button type="button" className="photo-open" aria-label={texts.photo} disabled={busy} onClick={onOpen}>
          <img className="photo-circle" src={preview} alt="" />
        </button>
      </span>
      <div className="photo-actions">
        <div className="photo-sources">{sources}</div>
        <button type="button" className="button" disabled={uploading} onClick={onRemove}>
          {texts.removePhoto}
        </button>
      </div>
    </div>
  )
}

function busyOrNot(busy: boolean): string {
  return busy ? 'photo-slot photo-slot-busy' : 'photo-slot'
}

function FieldHeader({ title, length, limit }: { title: string; length: number; limit: number }) {
  return (
    <span className="field-header">
      <span className="field-title">{title}</span>
      <span className="counter">
        {length}/{limit}
      </span>
    </span>
  )
}

function WarmthSlider({
  levels,
  value,
  onChange,
}: {
  levels: WarmthLevelOption[]
  value: number
  onChange: (value: number) => void
}) {
  // Слева снежинка, справа солнце - погода, под которую вещь надевают. Поэтому шкала
  // перевёрнута относительно уровней домена, где первый уровень самый лёгкий.
  const scale = [...levels].reverse()
  const position = Math.max(0, scale.findIndex((level) => level.value === value))
  const caption = scale[position].label

  function choose(index: number) {
    onChange(scale[index].value)
  }

  return (
    <div className="slider-field">
      <div className="slider-row">
        <span aria-hidden="true">{texts.warmthCold}</span>
        <input
          type="range"
          className="slider"
          min={0}
          max={scale.length - 1}
          step={1}
          value={position}
          aria-valuetext={caption}
          onChange={(event) => choose(Number(event.target.value))}
          onPointerUp={(event) => choose(Number(event.currentTarget.value))}
        />
        <span aria-hidden="true">{texts.warmthWarm}</span>
      </div>
      <p className="slider-caption">{caption}</p>
    </div>
  )
}

function sameValues(current: string[], expected: string[]): boolean {
  return current.length === expected.length && expected.every((value) => current.includes(value))
}

function toggle(values: string[], value: string): string[] {
  return values.includes(value) ? values.filter((current) => current !== value) : [...values, value]
}

function inOptionsOrder(options: Option[], selected: string[]): string[] {
  return options.map((option) => option.value).filter((value) => selected.includes(value))
}

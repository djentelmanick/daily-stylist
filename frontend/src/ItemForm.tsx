import { useState, type FormEvent } from 'react'
import type { ItemFields, Option, Options, WarmthLevelOption } from './api'
import { errorText, texts } from './texts'
import { Chip, Swatch } from './ui'

type Status = { kind: 'idle' } | { kind: 'submitting' } | { kind: 'failed'; message: string }

type Props = {
  options: Options
  title: string
  submitText: string
  initial?: ItemFields
  notice?: string
  onSubmit: (fields: ItemFields) => Promise<void>
}

export function ItemForm({ options, title, submitText, initial, notice, onSubmit }: Props) {
  const [name, setName] = useState(initial?.name ?? '')
  const [description, setDescription] = useState(initial?.description ?? '')
  const [category, setCategory] = useState(initial?.category ?? '')
  const [mainColor, setMainColor] = useState(initial?.main_color ?? '')
  const [extraColors, setExtraColors] = useState<string[]>(initial?.extra_colors ?? [])
  const [seasons, setSeasons] = useState<string[]>(initial?.seasons ?? [])
  const [warmthLevel, setWarmthLevel] = useState(initial?.warmth_level ?? 0)
  const [waterproof, setWaterproof] = useState(initial?.waterproof ?? false)
  const [status, setStatus] = useState<Status>({ kind: 'idle' })

  const needsWarmth = options.categories.find((option) => option.value === category)?.has_warmth ?? false
  const allSeasonsSelected = seasons.length === options.seasons.length
  const canSubmit =
    name.trim() !== '' &&
    category !== '' &&
    mainColor !== '' &&
    seasons.length > 0 &&
    (!needsWarmth || warmthLevel > 0) &&
    status.kind !== 'submitting'

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
        extra_colors: inOptionsOrder(options.colors, extraColors),
        seasons: inOptionsOrder(options.seasons, seasons),
        warmth_level: needsWarmth ? warmthLevel : 0,
        waterproof,
      })
      setStatus({ kind: 'idle' })
    } catch (error) {
      setStatus({ kind: 'failed', message: errorText(error) })
    }
  }

  return (
    <form className="screen" onSubmit={submit}>
      <h1>{title}</h1>

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
            <Chip key={option.value} selected={category === option.value} onClick={() => setCategory(option.value)}>
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
          <WarmthSlider levels={options.warmth_levels} value={warmthLevel} onChange={setWarmthLevel} />
        </fieldset>
      )}

      <label className="checkbox">
        <input type="checkbox" checked={waterproof} onChange={(event) => setWaterproof(event.target.checked)} />
        {texts.waterproof}
      </label>

      {notice && <p className="notice">{notice}</p>}
      {status.kind === 'failed' && <p className="notice notice-error">{status.message}</p>}

      <button type="submit" className="submit" disabled={!canSubmit}>
        {status.kind === 'submitting' ? texts.submitting : submitText}
      </button>
    </form>
  )
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
  const selectedIndex = levels.findIndex((level) => level.value === value)
  const chosen = selectedIndex !== -1
  const position = chosen ? selectedIndex : Math.floor((levels.length - 1) / 2)
  const caption = chosen ? levels[selectedIndex].label : texts.warmthNotChosen

  function choose(index: number) {
    onChange(levels[index].value)
  }

  return (
    <div className="slider-field">
      <input
        type="range"
        className={chosen ? 'slider' : 'slider slider-untouched'}
        min={0}
        max={levels.length - 1}
        step={1}
        value={position}
        aria-valuetext={caption}
        onChange={(event) => choose(Number(event.target.value))}
        onPointerUp={(event) => choose(Number(event.currentTarget.value))}
      />
      <div className="slider-scale">
        <span>{texts.warmthLighter}</span>
        <span>{texts.warmthWarmer}</span>
      </div>
      <p className={chosen ? 'slider-caption' : 'slider-caption slider-caption-empty'}>{caption}</p>
    </div>
  )
}

function toggle(values: string[], value: string): string[] {
  return values.includes(value) ? values.filter((current) => current !== value) : [...values, value]
}

function inOptionsOrder(options: Option[], selected: string[]): string[] {
  return options.map((option) => option.value).filter((value) => selected.includes(value))
}

import { useState, type ReactNode } from 'react'
import { changeItemStatus, deleteItems, labelOf, type Item, type Options } from './api'
import { confirm, useBackButton } from './telegram'
import { errorText, texts } from './texts'
import { Swatch } from './ui'

export function ItemScreen({
  options,
  item,
  onBack,
  onEdit,
  onChanged,
  onDeleted,
}: {
  options: Options
  item: Item
  onBack: () => void
  onEdit: () => void
  onChanged: (item: Item) => void
  onDeleted: () => void
}) {
  useBackButton(onBack)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function chooseStatus(status: string) {
    if (status === item.status || busy) {
      return
    }
    setBusy(true)
    setError('')
    try {
      await changeItemStatus(item.id, status)
      onChanged({ ...item, status })
    } catch (statusError) {
      setError(errorText(statusError))
    } finally {
      setBusy(false)
    }
  }

  async function remove() {
    if (!(await confirm(texts.confirmDeleteOne(item.name)))) {
      return
    }
    setBusy(true)
    setError('')
    try {
      await deleteItems([item.id])
      onDeleted()
    } catch (deleteError) {
      setError(errorText(deleteError))
      setBusy(false)
    }
  }

  const allSeasons = item.seasons.length === options.seasons.length

  return (
    <div className="screen">
      <header>
        <h1>{item.name}</h1>
        {item.description !== '' && <p className="hint">{item.description}</p>}
      </header>

      <section className="status-card">
        <h2 className="field-title">{texts.status}</h2>
        <div className="status-options">
          {options.statuses.map((option) => (
            <button
              key={option.value}
              type="button"
              className={option.value === item.status ? 'status-option status-option-selected' : 'status-option'}
              aria-pressed={option.value === item.status}
              disabled={busy}
              onClick={() => chooseStatus(option.value)}
            >
              {option.label}
            </button>
          ))}
        </div>
      </section>

      <dl className="properties">
        <Property title={texts.category}>{labelOf(options.categories, item.category)}</Property>
        <Property title={texts.colors}>
          <span className="color-list">
            {[item.main_color, ...item.extra_colors].map((color) => (
              <span key={color} className="color-list-item">
                <Swatch color={color} />
                {labelOf(options.colors, color)}
              </span>
            ))}
          </span>
        </Property>
        <Property title={texts.seasons}>
          {allSeasons ? texts.allSeasons : item.seasons.map((season) => labelOf(options.seasons, season)).join(', ')}
          {!item.seasons.includes(options.current_season) && (
            <span className="property-note">
              {texts.outOfSeasonNow(labelOf(options.seasons, options.current_season))}
            </span>
          )}
        </Property>
        {item.warmth_level > 0 && (
          <Property title={texts.warmthLevel}>{labelOf(options.warmth_levels, item.warmth_level)}</Property>
        )}
        <Property title={texts.waterproof}>{item.waterproof ? texts.yes : texts.no}</Property>
      </dl>

      {error !== '' && <p className="notice notice-error">{error}</p>}

      <div className="actions">
        <button type="button" className="button" disabled={busy} onClick={onEdit}>
          {texts.edit}
        </button>
        <button type="button" className="button button-danger" disabled={busy} onClick={remove}>
          {texts.delete}
        </button>
      </div>
    </div>
  )
}

function Property({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="property">
      <dt className="field-title">{title}</dt>
      <dd>{children}</dd>
    </div>
  )
}

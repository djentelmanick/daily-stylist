import { useEffect, useState } from 'react'
import { labelOf, reviewOutfit, type Item, type Options } from './api'
import { useBackButton } from './telegram'
import { errorText, texts } from './texts'
import { Icon, ItemMark, Notes, removeIcon, swapIcon } from './ui'

export function TodayOutfitScreen({
  options,
  items,
  onOpenItem,
  onReplace,
  onAdd,
  onRemove,
  onBack,
}: {
  options: Options
  items: Item[]
  onOpenItem: (itemId: number) => void
  onReplace: (itemId: number) => void
  onAdd: () => void
  onRemove: (itemId: number) => Promise<void>
  onBack: () => void
}) {
  useBackButton(onBack)
  const [reviewed, setReviewed] = useState<{ key: string; notes: string[] } | null>(null)
  const [removing, setRemoving] = useState(false)
  const [error, setError] = useState('')
  const key = items.map((item) => item.id).join(',')
  const notes = reviewed?.key === key ? reviewed.notes : []

  useEffect(() => {
    if (key === '') {
      return
    }
    let cancelled = false
    // Без заметок образ всё равно можно править, поэтому ошибку не показываем.
    reviewOutfit(key.split(',').map(Number))
      .then((result) => {
        if (!cancelled) {
          setReviewed({ key, notes: result })
        }
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [key])

  async function remove(itemId: number) {
    setRemoving(true)
    setError('')
    try {
      await onRemove(itemId)
    } catch (removeError) {
      setError(errorText(removeError))
    } finally {
      setRemoving(false)
    }
  }

  return (
    <div className="screen">
      <h1>{texts.todayTitle}</h1>

      {items.length === 0 ? (
        <p className="hint">{texts.todayEmpty}</p>
      ) : (
        <ul className="item-list">
          {items.map((item) => (
            <li key={item.id} className="item-row-actions">
              <button type="button" className="item-row" onClick={() => onOpenItem(item.id)}>
                <ItemMark item={item} />
                <span className="item-row-text">
                  <span className="item-row-name">{item.name}</span>
                  <span className="hint">{labelOf(options.categories, item.category)}</span>
                </span>
              </button>
              <button
                type="button"
                className="icon-button"
                aria-label={texts.replaceItem}
                disabled={removing}
                onClick={() => onReplace(item.id)}
              >
                <Icon path={swapIcon} />
              </button>
              <button
                type="button"
                className="icon-button"
                aria-label={texts.removeItem}
                disabled={removing}
                onClick={() => remove(item.id)}
              >
                <Icon path={removeIcon} />
              </button>
            </li>
          ))}
        </ul>
      )}

      <Notes notes={notes} />
      {error !== '' && <p className="notice notice-error">{error}</p>}

      <button type="button" className="submit" disabled={removing} onClick={onAdd}>
        {texts.addToOutfit}
      </button>
    </div>
  )
}

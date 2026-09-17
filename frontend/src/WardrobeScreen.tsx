import { useState } from 'react'
import { availableStatus, deleteItems, labelOf, type Item, type Options } from './api'
import { confirm, useBackButton, vibrate } from './telegram'
import { errorText, texts } from './texts'
import { ItemMark } from './ui'
import { useLongPress } from './useLongPress'

export function WardrobeScreen({
  options,
  items,
  wornItemIds,
  onBack,
  onOpenItem,
  onAddItem,
  onDeleted,
}: {
  options: Options
  items: Item[]
  wornItemIds: number[]
  onBack: () => void
  onOpenItem: (itemId: number) => void
  onAddItem: () => void
  onDeleted: (itemIds: number[]) => void
}) {
  // null - обычный просмотр, массив - режим выбора, даже если ничего не выбрано.
  const [selected, setSelected] = useState<number[] | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [error, setError] = useState('')

  useBackButton(selected === null ? onBack : () => setSelected(null))

  function toggle(itemId: number) {
    setSelected((current) =>
      current?.includes(itemId) ? current.filter((id) => id !== itemId) : [...(current ?? []), itemId],
    )
  }

  function select(itemId: number) {
    vibrate()
    toggle(itemId)
  }

  async function deleteSelected() {
    if (selected === null || selected.length === 0) {
      return
    }
    const question =
      selected.length === 1
        ? texts.confirmDeleteOne(items.find((item) => item.id === selected[0])?.name ?? '')
        : texts.confirmDeleteMany(selected.length)
    if (!(await confirm(question))) {
      return
    }

    setDeleting(true)
    setError('')
    try {
      await deleteItems(selected)
      onDeleted(selected)
      setSelected(null)
    } catch (deleteError) {
      setError(errorText(deleteError))
    } finally {
      setDeleting(false)
    }
  }

  if (items.length === 0) {
    return (
      <div className="screen">
        <h1>{texts.wardrobe}</h1>
        <p className="hint">{texts.emptyWardrobe}</p>
        <button type="button" className="submit" onClick={onAddItem}>
          {texts.addItem}
        </button>
      </div>
    )
  }

  return (
    <div className="screen">
      <header className="screen-header">
        <h1>{selected === null ? texts.wardrobe : texts.selectedCount(selected.length)}</h1>
        <button type="button" className="link-button" onClick={() => setSelected(selected === null ? [] : null)}>
          {selected === null ? texts.select : texts.cancel}
        </button>
      </header>
      {selected === null && <p className="hint">{texts.selectHint}</p>}

      <ul className="item-list">
        {items.map((item) => (
          <ItemRow
            key={item.id}
            item={item}
            categoryLabel={labelOf(options.categories, item.category)}
            statusLabel={labelOf(options.statuses, item.status)}
            inSeason={item.seasons.includes(options.current_season)}
            worn={wornItemIds.includes(item.id)}
            checked={selected?.includes(item.id) ?? null}
            onPress={() => (selected === null ? onOpenItem(item.id) : select(item.id))}
            onLongPress={() => select(item.id)}
          />
        ))}
      </ul>

      {error !== '' && <p className="notice notice-error">{error}</p>}

      {selected === null ? (
        <button type="button" className="submit" onClick={onAddItem}>
          {texts.addItem}
        </button>
      ) : (
        <button
          type="button"
          className="submit submit-danger"
          disabled={selected.length === 0 || deleting}
          onClick={deleteSelected}
        >
          {deleting ? texts.deleting : texts.deleteSelected(selected.length)}
        </button>
      )}
    </div>
  )
}

function ItemRow({
  item,
  categoryLabel,
  statusLabel,
  inSeason,
  worn,
  checked,
  onPress,
  onLongPress,
}: {
  item: Item
  categoryLabel: string
  statusLabel: string
  inSeason: boolean
  worn: boolean
  // null - не в режиме выбора.
  checked: boolean | null
  onPress: () => void
  onLongPress: () => void
}) {
  const press = useLongPress(onLongPress, onPress)

  return (
    <li>
      <button
        type="button"
        className={checked ? 'item-row item-row-checked' : 'item-row'}
        aria-pressed={checked ?? undefined}
        {...press}
      >
        {checked !== null && <span className={checked ? 'checkmark checkmark-on' : 'checkmark'}>{checked && '✓'}</span>}
        <ItemMark item={item} />
        <span className="item-row-text">
          <span className="item-row-name">{item.name}</span>
          <span className="hint">{categoryLabel}</span>
        </span>
        {(worn || item.status !== availableStatus || !inSeason) && (
          <span className="badges">
            {worn && <span className="badge badge-worn">{texts.worn}</span>}
            {item.status !== availableStatus && <span className="badge">{statusLabel}</span>}
            {!inSeason && <span className="badge">{texts.notInSeason}</span>}
          </span>
        )}
      </button>
    </li>
  )
}

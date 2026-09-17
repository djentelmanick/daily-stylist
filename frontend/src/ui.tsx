import type { ReactNode } from 'react'
import type { Item } from './api'
import { swatch } from './colors'

export function Chip({ selected, onClick, children }: { selected: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button type="button" className={selected ? 'chip chip-selected' : 'chip'} aria-pressed={selected} onClick={onClick}>
      {children}
    </button>
  )
}

export function Swatch({ color }: { color: string }) {
  return <span className="swatch" style={{ background: swatch(color) }} />
}

export function ItemAvatar({ item }: { item: Item }) {
  if (item.photo_url !== '') {
    return <img className="item-avatar" src={item.photo_url} alt="" loading="lazy" />
  }
  return <Swatch color={item.main_color} />
}

const maxMarkColors = 3

export function ItemMark({ item }: { item: Item }) {
  if (item.photo_url !== '') {
    return <img className="item-thumb" src={item.photo_url} alt="" loading="lazy" />
  }
  return (
    <span className="item-colors">
      <span className="item-color-main" style={{ background: swatch(item.main_color) }} />
      {item.extra_colors.length > 0 && (
        <span className="item-color-extras">
          {item.extra_colors.slice(0, maxMarkColors).map((color) => (
            <span key={color} className="item-color-extra" style={{ background: swatch(color) }} />
          ))}
        </span>
      )}
    </span>
  )
}

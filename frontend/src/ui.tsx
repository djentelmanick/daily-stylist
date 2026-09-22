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

// Многоточие в тексте не пишем - точки уже здесь.
export function Thinking({ children }: { children: string }) {
  return (
    <span className="thinking">
      <span className="thinking-text">{children}</span>
      <Dots />
    </span>
  )
}

// На кнопке блик по буквам не читается, поэтому там только точки.
export function Dots() {
  return (
    <span className="thinking-dots" aria-hidden="true">
      <i />
      <i />
      <i />
    </span>
  )
}

export function Chevron({ direction }: { direction: 'left' | 'right' }) {
  return (
    <svg
      className="icon"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d={direction === 'left' ? 'M15 6l-6 6 6 6' : 'M9 6l6 6-6 6'} />
    </svg>
  )
}

export function Icon({ path }: { path: string }) {
  return (
    <svg
      className="icon"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d={path} />
    </svg>
  )
}

export const swapIcon = 'M7 4L3 8l4 4M3 8h14M17 12l4 4-4 4M21 16H7'
export const removeIcon = 'M6 6l12 12M18 6L6 18'

export function Notes({ notes }: { notes: string[] }) {
  if (notes.length === 0) {
    return null
  }
  return (
    <ul className="notes">
      {notes.map((note) => (
        <li key={note}>{note}</li>
      ))}
    </ul>
  )
}

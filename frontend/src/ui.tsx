import type { ReactNode } from 'react'
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

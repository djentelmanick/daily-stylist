import { useEffect, useState } from 'react'
import { fetchCandidates, labelOf, type Candidate, type Item, type Options } from './api'
import { useBackButton } from './telegram'
import { errorText, texts } from './texts'
import { ItemMark, Thinking } from './ui'

type State =
  | { kind: 'loading' }
  | { kind: 'failed'; message: string }
  | { kind: 'ready'; candidates: Candidate[] }

export function PickItemScreen({
  options,
  items,
  outfit,
  replace,
  onPick,
  onBack,
}: {
  options: Options
  items: Item[]
  outfit: number[]
  replace: number | null
  onPick: (item: Item) => Promise<void>
  onBack: () => void
}) {
  useBackButton(onBack)
  const [state, setState] = useState<State>({ kind: 'loading' })
  const [query, setQuery] = useState('')
  const [picking, setPicking] = useState(false)
  const [error, setError] = useState('')
  const replaced = items.find((item) => item.id === replace)

  useEffect(() => {
    let cancelled = false
    fetchCandidates(outfit, replace)
      .then((candidates) => {
        if (!cancelled) {
          setState({ kind: 'ready', candidates })
        }
      })
      .catch((fetchError: unknown) => {
        if (!cancelled) {
          setState({ kind: 'failed', message: errorText(fetchError) })
        }
      })
    return () => {
      cancelled = true
    }
  }, [outfit, replace])

  async function pick(item: Item) {
    setPicking(true)
    setError('')
    try {
      await onPick(item)
    } catch (pickError) {
      setError(errorText(pickError))
      setPicking(false)
    }
  }

  const words = normalize(query).split(/\s+/).filter((word) => word !== '')
  const found =
    state.kind === 'ready'
      ? state.candidates.flatMap((candidate) => {
          const item = items.find((one) => one.id === candidate.item_id)
          if (item === undefined) {
            return []
          }
          const text = normalize(`${item.name} ${item.description} ${labelOf(options.categories, item.category)}`)
          return words.every((word) => text.includes(word)) ? [{ item, notes: candidate.notes }] : []
        })
      : []

  return (
    <div className="screen">
      <header>
        <h1>{replaced === undefined ? texts.addToOutfit : texts.replaceTitle(replaced.name)}</h1>
        <p className="hint">{texts.pickHint}</p>
      </header>

      <input
        type="text"
        value={query}
        placeholder={texts.searchPlaceholder}
        enterKeyHint="search"
        onChange={(event) => setQuery(event.target.value)}
      />

      {state.kind === 'loading' && (
        <p className="hint">
          <Thinking>{texts.searchingCandidates}</Thinking>
        </p>
      )}
      {state.kind === 'failed' && <p className="notice notice-error">{state.message}</p>}
      {state.kind === 'ready' && found.length === 0 && (
        <p className="hint">{words.length > 0 ? texts.nothingFound : texts.nothingToPick}</p>
      )}
      {found.length > 0 && (
        <ul className="item-list">
          {found.map(({ item, notes }) => {
            const hints = replaced === undefined ? [labelOf(options.categories, item.category), ...notes] : notes
            return (
              <li key={item.id}>
                <button type="button" className="item-row" disabled={picking} onClick={() => pick(item)}>
                  <ItemMark item={item} />
                  <span className="item-row-text">
                    <span className="item-row-name">{item.name}</span>
                    {hints.length > 0 && <span className="hint">{hints.join(' · ')}</span>}
                  </span>
                </button>
              </li>
            )
          })}
        </ul>
      )}

      {error !== '' && <p className="notice notice-error">{error}</p>}
    </div>
  )
}

function normalize(text: string): string {
  return text.toLowerCase().replaceAll('ё', 'е')
}

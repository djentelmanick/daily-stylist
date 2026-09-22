import { useEffect, useState } from 'react'
import { ApiError, fetchRecommendation, labelOf, wearToday, type Options, type Recommendation } from './api'
import { useBackButton } from './telegram'
import { errorText, texts } from './texts'
import { Chevron, Dots, Icon, ItemMark, Notes, swapIcon, Thinking } from './ui'

export type RecommendationMemory = {
  recommendation: Recommendation | null
  index: number
}

type State =
  | { kind: 'loading' }
  | { kind: 'needsCity' }
  | { kind: 'failed'; message: string }
  | { kind: 'ready'; recommendation: Recommendation }

export function RecommendationScreen({
  options,
  memory,
  wornItemIds,
  onWorn,
  onBack,
  onOpenItem,
  onReplaceItem,
  onChooseCity,
  onAddItem,
}: {
  options: Options
  memory: RecommendationMemory
  wornItemIds: number[]
  onWorn: (itemIds: number[]) => void
  onBack: () => void
  onOpenItem: (itemId: number) => void
  onReplaceItem: (itemId: number, outfit: number[]) => void
  onChooseCity: () => void
  onAddItem: () => void
}) {
  useBackButton(onBack)
  const [state, setState] = useState<State>(() =>
    memory.recommendation === null ? { kind: 'loading' } : { kind: 'ready', recommendation: memory.recommendation },
  )

  useEffect(() => {
    if (memory.recommendation !== null) {
      return
    }

    let cancelled = false
    fetchRecommendation()
      .then((recommendation) => {
        memory.recommendation = recommendation
        if (!cancelled) {
          setState({ kind: 'ready', recommendation })
        }
      })
      .catch((error: unknown) => {
        if (cancelled) {
          return
        }
        const needsCity = error instanceof ApiError && error.code === 'location_not_set'
        setState(needsCity ? { kind: 'needsCity' } : { kind: 'failed', message: errorText(error) })
      })
    return () => {
      cancelled = true
    }
  }, [memory])

  switch (state.kind) {
    case 'loading':
      return (
        <p className="message">
          <Thinking>{texts.pickingOutfit}</Thinking>
        </p>
      )
    case 'failed':
      return <p className="message">{state.message}</p>
    case 'needsCity':
      return (
        <div className="screen">
          <h1>{texts.recommendation}</h1>
          <p className="notice">{texts.cityNeeded}</p>
          <button type="button" className="submit" onClick={onChooseCity}>
            {texts.chooseCity}
          </button>
        </div>
      )
    case 'ready':
      return (
        <RecommendationView
          options={options}
          recommendation={state.recommendation}
          memory={memory}
          wornItemIds={wornItemIds}
          onWorn={onWorn}
          onOpenItem={onOpenItem}
          onReplaceItem={onReplaceItem}
          onChooseCity={onChooseCity}
          onAddItem={onAddItem}
        />
      )
  }
}

function RecommendationView({
  options,
  recommendation,
  memory,
  wornItemIds,
  onWorn,
  onOpenItem,
  onReplaceItem,
  onChooseCity,
  onAddItem,
}: {
  options: Options
  recommendation: Recommendation
  memory: RecommendationMemory
  wornItemIds: number[]
  onWorn: (itemIds: number[]) => void
  onOpenItem: (itemId: number) => void
  onReplaceItem: (itemId: number, outfit: number[]) => void
  onChooseCity: () => void
  onAddItem: () => void
}) {
  const { city, weather, outfits } = recommendation
  const [index, setIndex] = useState(memory.index)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const outfit = outfits.at(index)
  const worn = outfit !== undefined && sameItems(outfit.items.map((item) => item.id), wornItemIds)

  function show(step: number) {
    const next = (index + step + outfits.length) % outfits.length
    memory.index = next
    setIndex(next)
    setError('')
  }

  async function wear() {
    if (outfit === undefined) {
      return
    }
    setSaving(true)
    setError('')
    try {
      const itemIds = outfit.items.map((item) => item.id)
      await wearToday(itemIds)
      onWorn(itemIds)
    } catch (wearError) {
      setError(errorText(wearError))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="screen">
      <h1>{texts.recommendation}</h1>

      <section className="weather-card">
        <div className="weather-city">
          <span className="hint">{city.name}</span>
          <button type="button" className="link-button" onClick={onChooseCity}>
            {texts.changeCity}
          </button>
        </div>
        <p className="weather-temperature">{weather.temperature}</p>
        {weather.details.length > 0 && <p className="hint">{weather.details.join(' · ')}</p>}
      </section>

      {outfit === undefined ? (
        <>
          <Notes notes={recommendation.notes} />
          <button type="button" className="submit" onClick={onAddItem}>
            {texts.addItem}
          </button>
        </>
      ) : (
        <>
          <section className="outfit">
            {outfits.length > 1 && (
              <div className="outfit-pager">
                <p className="hint">{texts.outfitNumber(index + 1, outfits.length)}</p>
                <div className="outfit-arrows">
                  <button
                    type="button"
                    className="icon-button"
                    aria-label={texts.previousOutfit}
                    disabled={saving}
                    onClick={() => show(-1)}
                  >
                    <Chevron direction="left" />
                  </button>
                  <button
                    type="button"
                    className="icon-button"
                    aria-label={texts.nextOutfit}
                    disabled={saving}
                    onClick={() => show(1)}
                  >
                    <Chevron direction="right" />
                  </button>
                </div>
              </div>
            )}
            <ul className="item-list">
              {outfit.items.map((item) => (
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
                    disabled={saving}
                    onClick={() =>
                      onReplaceItem(
                        item.id,
                        outfit.items.map((one) => one.id),
                      )
                    }
                  >
                    <Icon path={swapIcon} />
                  </button>
                </li>
              ))}
            </ul>
          </section>

          <Notes notes={outfit.notes} />
          {error !== '' && <p className="notice notice-error">{error}</p>}

          {worn && <p className="hint">{texts.wornHint}</p>}

          <div className="actions">
            <button type="button" className="button button-accent" disabled={saving || worn} onClick={wear}>
              {worn ? (
                texts.worn
              ) : saving ? (
                <>
                  {texts.submitting}
                  <Dots />
                </>
              ) : (
                texts.wear
              )}
            </button>
          </div>
        </>
      )}
    </div>
  )
}

function sameItems(itemIds: number[], otherIds: number[]): boolean {
  return itemIds.length === otherIds.length && itemIds.every((itemId) => otherIds.includes(itemId))
}

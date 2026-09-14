import { useEffect, useState } from 'react'
import { ApiError, fetchRecommendation, labelOf, wearToday, type Options, type Recommendation } from './api'
import { useBackButton } from './telegram'
import { errorText, texts } from './texts'
import { Swatch } from './ui'

type State =
  | { kind: 'loading' }
  | { kind: 'needsCity' }
  | { kind: 'failed'; message: string }
  | { kind: 'ready'; recommendation: Recommendation }

export function RecommendationScreen({
  options,
  wornItemIds,
  onWorn,
  onBack,
  onChooseCity,
  onAddItem,
}: {
  options: Options
  wornItemIds: number[]
  onWorn: (itemIds: number[]) => void
  onBack: () => void
  onChooseCity: () => void
  onAddItem: () => void
}) {
  useBackButton(onBack)
  const [state, setState] = useState<State>({ kind: 'loading' })

  // Экран открывается заново после выбора города и после «Назад», поэтому образ всегда свежий.
  useEffect(() => {
    let cancelled = false
    fetchRecommendation()
      .then((recommendation) => {
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
  }, [])

  switch (state.kind) {
    case 'loading':
      return <p className="message">{texts.pickingOutfit}</p>
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
          wornItemIds={wornItemIds}
          onWorn={onWorn}
          onChooseCity={onChooseCity}
          onAddItem={onAddItem}
        />
      )
  }
}

function RecommendationView({
  options,
  recommendation,
  wornItemIds,
  onWorn,
  onChooseCity,
  onAddItem,
}: {
  options: Options
  recommendation: Recommendation
  wornItemIds: number[]
  onWorn: (itemIds: number[]) => void
  onChooseCity: () => void
  onAddItem: () => void
}) {
  const { city, weather, outfits } = recommendation
  const [index, setIndex] = useState(0)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const outfit = outfits.at(index)
  const worn = outfit !== undefined && sameItems(outfit.items.map((item) => item.id), wornItemIds)

  function showNext() {
    setIndex((index + 1) % outfits.length)
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
            {outfits.length > 1 && <p className="hint">{texts.outfitNumber(index + 1, outfits.length)}</p>}
            <ul className="item-list">
              {outfit.items.map((item) => (
                <li key={item.id} className="item-row item-row-static">
                  <span className="swatches">
                    {[item.main_color, ...item.extra_colors].map((color) => (
                      <Swatch key={color} color={color} />
                    ))}
                  </span>
                  <span className="item-row-text">
                    <span className="item-row-name">{item.name}</span>
                    <span className="hint">{labelOf(options.categories, item.category)}</span>
                  </span>
                </li>
              ))}
            </ul>
          </section>

          <Notes notes={outfit.notes} />
          {error !== '' && <p className="notice notice-error">{error}</p>}

          <div className="actions">
            {outfits.length > 1 && (
              <button type="button" className="button" disabled={saving} onClick={showNext}>
                {texts.anotherOutfit}
              </button>
            )}
            <button type="button" className="button button-accent" disabled={saving || worn} onClick={wear}>
              {worn ? texts.worn : saving ? texts.submitting : texts.wear}
            </button>
          </div>
          {worn && <p className="hint">{texts.wornHint}</p>}
        </>
      )}
    </div>
  )
}

function sameItems(itemIds: number[], otherIds: number[]): boolean {
  return itemIds.length === otherIds.length && itemIds.every((itemId) => otherIds.includes(itemId))
}

function Notes({ notes }: { notes: string[] }) {
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

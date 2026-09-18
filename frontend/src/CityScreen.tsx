import { useEffect, useState } from 'react'
import { saveCity, searchCities, type City } from './api'
import { useBackButton } from './telegram'
import { errorText, texts } from './texts'
import { Thinking } from './ui'

const minQueryLength = 2
const searchDelayMs = 300

export function CityScreen({ onBack, onSaved }: { onBack: () => void; onSaved: () => void }) {
  useBackButton(onBack)
  const [query, setQuery] = useState('')
  const [found, setFound] = useState<{ query: string; cities: City[] } | null>(null)
  const [failed, setFailed] = useState<{ query: string; message: string } | null>(null)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')

  const trimmed = query.trim()
  const searchable = trimmed.length >= minQueryLength
  const cities = searchable && found?.query === trimmed ? found.cities : null
  const searchError = searchable && failed?.query === trimmed ? failed.message : ''
  const searching = searchable && cities === null && searchError === ''

  useEffect(() => {
    if (!searchable) {
      return
    }
    let cancelled = false
    const timer = window.setTimeout(() => {
      searchCities(trimmed)
        .then((result) => {
          if (!cancelled) {
            setFound({ query: trimmed, cities: result })
          }
        })
        .catch((error: unknown) => {
          if (!cancelled) {
            setFailed({ query: trimmed, message: errorText(error) })
          }
        })
    }, searchDelayMs)
    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [searchable, trimmed])

  async function choose(city: City) {
    setSaving(true)
    setSaveError('')
    try {
      await saveCity(city)
      onSaved()
    } catch (error) {
      setSaveError(errorText(error))
      setSaving(false)
    }
  }

  return (
    <div className="screen">
      <header>
        <h1>{texts.cityTitle}</h1>
        <p className="hint">{texts.cityHint}</p>
      </header>

      <input
        type="text"
        value={query}
        placeholder={texts.cityPlaceholder}
        enterKeyHint="search"
        autoFocus
        onChange={(event) => setQuery(event.target.value)}
      />

      {searching && (
        <p className="hint">
          <Thinking>{texts.citySearching}</Thinking>
        </p>
      )}
      {cities?.length === 0 && <p className="hint">{texts.cityNotFound}</p>}
      {cities !== null && cities.length > 0 && (
        <ul className="item-list">
          {cities.map((city) => (
            <li key={`${city.latitude},${city.longitude}`}>
              <button type="button" className="item-row" disabled={saving} onClick={() => choose(city)}>
                <span className="item-row-text">
                  <span className="item-row-name">{city.name}</span>
                  {city.region !== '' && <span className="hint">{city.region}</span>}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}

      {searchError !== '' && <p className="notice notice-error">{searchError}</p>}
      {saveError !== '' && <p className="notice notice-error">{saveError}</p>}
    </div>
  )
}

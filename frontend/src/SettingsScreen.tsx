import { useEffect, useRef, useState } from 'react'
import { fetchSettings, saveSettings, type Settings } from './api'
import { useBackButton } from './telegram'
import { errorText, texts } from './texts'
import { Thinking } from './ui'

// Время сохраняется не на каждую цифру: иначе запрос уходит после первой же,
// а поле перерисовывается прямо посреди набора.
const saveDelayMs = 800

export function SettingsScreen({ onBack, onChooseCity }: { onBack: () => void; onChooseCity: () => void }) {
  useBackButton(onBack)
  const [settings, setSettings] = useState<Settings | null>(null)
  const [sendAt, setSendAt] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const unsaved = useRef<Settings | null>(null)

  // Уход с экрана раньше, чем сработает задержка, не должен терять правку.
  useEffect(
    () => () => {
      if (unsaved.current !== null) {
        const { morning_enabled, send_at } = unsaved.current
        void saveSettings({ morning_enabled, send_at })
      }
    },
    [],
  )

  useEffect(() => {
    let cancelled = false
    fetchSettings()
      .then((loaded) => {
        if (!cancelled) {
          setSettings(loaded)
          setSendAt(loaded.send_at)
        }
      })
      .catch((failure: unknown) => {
        if (!cancelled) {
          setError(errorText(failure))
        }
      })
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    if (settings === null || sendAt === '' || sendAt === settings.send_at) {
      return
    }
    const changed = { ...settings, send_at: sendAt }
    unsaved.current = changed
    const timer = window.setTimeout(() => void save(changed), saveDelayMs)
    return () => window.clearTimeout(timer)
  }, [sendAt, settings])

  async function save(changed: Settings) {
    const previous = settings
    setSettings(changed)
    setSaving(true)
    setError('')
    try {
      await saveSettings({ morning_enabled: changed.morning_enabled, send_at: changed.send_at })
      unsaved.current = null
    } catch (failure) {
      setSettings(previous)
      setSendAt(previous?.send_at ?? '')
      setError(errorText(failure))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="screen">
      <h1>{texts.settingsTitle}</h1>

      {settings === null ? (
        error === '' && (
          <p className="hint">
            <Thinking>{texts.loading}</Thinking>
          </p>
        )
      ) : (
        <>
          <section className="status-card">
            <h2 className="field-title">{texts.morningTitle}</h2>
            <div className="status-options">
              {[true, false].map((enabled) => (
                <button
                  key={String(enabled)}
                  type="button"
                  className={
                    enabled === settings.morning_enabled ? 'status-option status-option-selected' : 'status-option'
                  }
                  aria-pressed={enabled === settings.morning_enabled}
                  disabled={saving}
                  onClick={() => enabled !== settings.morning_enabled && save({ ...settings, morning_enabled: enabled })}
                >
                  {enabled ? texts.morningOn : texts.morningOff}
                </button>
              ))}
            </div>

            {settings.morning_enabled ? (
              <label className="field">
                <span className="field-title">{texts.sendAt}</span>
                <input type="time" value={sendAt} onChange={(event) => setSendAt(event.target.value)} />
              </label>
            ) : (
              <p className="hint">{texts.morningOffHint}</p>
            )}
          </section>

          <section className="status-card">
            <h2 className="field-title">{texts.city}</h2>
            <div className="card-row">
              <span>{settings.city?.name ?? texts.cityNotChosen}</span>
              <button type="button" className="link-button" onClick={onChooseCity}>
                {settings.city === null ? texts.chooseCity : texts.changeCity}
              </button>
            </div>
            <p className="hint">{texts.cityHint}</p>
          </section>
        </>
      )}

      {error !== '' && <p className="notice notice-error">{error}</p>}
    </div>
  )
}

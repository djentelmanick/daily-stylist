import { useEffect, useState, type ReactNode } from 'react'
import { fetchSettings, labelOf, type City, type Item, type Options } from './api'
import { CityScreen } from './CityScreen'
import { MorningCard, useSettings } from './SettingsScreen'
import { useBackButton } from './telegram'
import { texts } from './texts'
import { ItemMark, Thinking } from './ui'

export type OnboardingStep = 'city' | 'wardrobe' | 'morning'

const steps: OnboardingStep[] = ['city', 'wardrobe', 'morning']

// Без них подбор не соберёт образ. Платье и комбинезон заменяют и верх, и низ.
const basics: { category: string; covered: string[] }[] = [
  { category: 'top', covered: ['top', 'dress', 'jumpsuit'] },
  { category: 'bottom', covered: ['bottom', 'dress', 'jumpsuit'] },
  { category: 'shoes', covered: ['shoes'] },
  { category: 'outerwear', covered: ['outerwear'] },
]

function covers(items: Item[], covered: string[]): boolean {
  return items.some((item) => covered.includes(item.category))
}

export function hasBasics(items: Item[]): boolean {
  return basics.every(({ covered }) => covers(items, covered))
}

export function OnboardingScreen({
  options,
  step,
  items,
  onStep,
  onAddItem,
  onOpenItem,
  onFinish,
  onClose,
}: {
  options: Options
  step: OnboardingStep
  items: Item[]
  onStep: (step: OnboardingStep) => void
  onAddItem: () => void
  onOpenItem: (itemId: number) => void
  onFinish: () => void
  onClose: () => void
}) {
  const index = steps.indexOf(step)
  const last = index === steps.length - 1
  const back = () => (index > 0 ? onStep(steps[index - 1]) : onClose())
  const next = () => (last ? onFinish() : onStep(steps[index + 1]))

  const header = (title: string, hint: string) => (
    <header>
      <div className="card-row">
        <p className="hint">{texts.onboardingStep(index + 1, steps.length)}</p>
        <button type="button" className="link-button" onClick={onClose}>
          {texts.skip}
        </button>
      </div>
      <h1>{title}</h1>
      <p className="hint">{hint}</p>
    </header>
  )
  const footer = (canGoNext: boolean) => (
    <div className="actions">
      {index > 0 && (
        <button type="button" className="button" onClick={back}>
          {texts.back}
        </button>
      )}
      <button type="button" className="button button-accent" disabled={!canGoNext} onClick={next}>
        {last ? texts.showOutfit : texts.next}
      </button>
    </div>
  )

  switch (step) {
    case 'city':
      return <CityStep options={options} header={header} footer={footer} onBack={back} />
    case 'wardrobe':
      return (
        <WardrobeStep
          options={options}
          items={items}
          header={header(texts.onboardingWardrobeTitle, texts.onboardingWardrobeHint)}
          footer={footer(items.length > 0)}
          onAddItem={onAddItem}
          onOpenItem={onOpenItem}
          onBack={back}
        />
      )
    case 'morning':
      return (
        <MorningStep
          header={header(texts.onboardingMorningTitle, texts.onboardingMorningHint)}
          footer={footer(true)}
          onBack={back}
        />
      )
  }
}

function CityStep({
  options,
  header,
  footer,
  onBack,
}: {
  options: Options
  header: (title: string, hint: string) => ReactNode
  footer: (canGoNext: boolean) => ReactNode
  onBack: () => void
}) {
  // undefined - ещё не знаем, null - город не выбран.
  const [city, setCity] = useState<City | null | undefined>(undefined)

  useEffect(() => {
    let cancelled = false
    fetchSettings()
      .then((settings) => !cancelled && setCity(settings.city))
      .catch(() => !cancelled && setCity(null))
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <CityScreen
      minQueryLength={options.limits.min_city_query}
      header={
        <>
          {header(texts.onboardingCityTitle, texts.onboardingCityHint)}
          {city != null && (
            <section className="status-card">
              <h2 className="field-title">{texts.city}</h2>
              <span>{city.region === '' ? city.name : `${city.name}, ${city.region}`}</span>
              <p className="hint">{texts.onboardingCityChosen}</p>
            </section>
          )}
        </>
      }
      footer={footer(city != null)}
      onBack={onBack}
      onSaved={setCity}
    />
  )
}

function WardrobeStep({
  options,
  items,
  header,
  footer,
  onAddItem,
  onOpenItem,
  onBack,
}: {
  options: Options
  items: Item[]
  header: ReactNode
  footer: ReactNode
  onAddItem: () => void
  onOpenItem: (itemId: number) => void
  onBack: () => void
}) {
  useBackButton(onBack)

  return (
    <div className="screen">
      {header}

      <section className="status-card">
        <h2 className="field-title">{texts.onboardingBasics}</h2>
        <ul className="basics">
          {basics.map(({ category, covered }) => {
            const done = covers(items, covered)
            return (
              <li key={category}>
                <span className={done ? 'checkmark checkmark-on' : 'checkmark'}>{done && '✓'}</span>
                {labelOf(options.categories, category)}
              </li>
            )
          })}
        </ul>
      </section>

      {items.length > 0 && (
        <>
          <p className="hint">{texts.onboardingAdded(items.length)}</p>
          <ul className="item-list">
            {items.map((item) => (
              <li key={item.id}>
                <button type="button" className="item-row" onClick={() => onOpenItem(item.id)}>
                  <ItemMark item={item} />
                  <span className="item-row-text">
                    <span className="item-row-name">{item.name}</span>
                    <span className="hint">{labelOf(options.categories, item.category)}</span>
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </>
      )}

      <button type="button" className="submit" onClick={onAddItem}>
        {texts.addItem}
      </button>
      {footer}
    </div>
  )
}

function MorningStep({ header, footer, onBack }: { header: ReactNode; footer: ReactNode; onBack: () => void }) {
  useBackButton(onBack)
  const state = useSettings()
  const { settings, error } = state

  return (
    <div className="screen">
      {header}

      {settings === null ? (
        error === '' && (
          <p className="hint">
            <Thinking>{texts.loading}</Thinking>
          </p>
        )
      ) : (
        <MorningCard {...state} settings={settings} />
      )}
      {error !== '' && <p className="notice notice-error">{error}</p>}

      {footer}
    </div>
  )
}

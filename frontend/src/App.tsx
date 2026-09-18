import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { fetchItems, fetchOptions, fetchTodayOutfit, type Item, type Options } from './api'
import { CityScreen } from './CityScreen'
import { HomeScreen } from './HomeScreen'
import { AddItemScreen, EditItemScreen } from './ItemFormScreens'
import { ItemScreen, PhotoScreen } from './ItemScreen'
import { RecommendationScreen, type RecommendationMemory } from './RecommendationScreen'
import { recall, remember } from './session'
import { getInitData, useBackButton } from './telegram'
import { errorTexts, texts } from './texts'
import { WardrobeScreen } from './WardrobeScreen'
import { Thinking } from './ui'

type State =
  | { kind: 'loading' }
  | { kind: 'outsideTelegram' }
  | { kind: 'failed' }
  | { kind: 'ready'; options: Options; items: Item[]; todayOutfit: number[] }

export function App() {
  const [state, setState] = useState<State>(() =>
    getInitData() === '' ? { kind: 'outsideTelegram' } : { kind: 'loading' },
  )

  useEffect(() => {
    if (state.kind !== 'loading') {
      return
    }

    let cancelled = false
    Promise.all([fetchOptions(), fetchItems(), fetchTodayOutfit()])
      .then(([options, items, todayOutfit]) => {
        if (!cancelled) {
          setState({ kind: 'ready', options, items, todayOutfit })
        }
      })
      .catch(() => {
        if (!cancelled) {
          setState({ kind: 'failed' })
        }
      })
    return () => {
      cancelled = true
    }
  }, [state.kind])

  switch (state.kind) {
    case 'loading':
      return (
        <p className="message">
          <Thinking>{texts.loading}</Thinking>
        </p>
      )
    case 'outsideTelegram':
      return <p className="message">{texts.openFromBot}</p>
    case 'failed':
      return <p className="message">{texts.loadFailed}</p>
    case 'ready':
      return <Screens options={state.options} initialItems={state.items} initialTodayOutfit={state.todayOutfit} />
  }
}

type Screen =
  | { kind: 'home' }
  | { kind: 'wardrobe' }
  | { kind: 'item'; itemId: number }
  | { kind: 'edit'; itemId: number }
  | { kind: 'photo'; itemId: number }
  | { kind: 'draftPhoto'; url: string; caption: string }
  | { kind: 'add' }
  | { kind: 'recommendation' }
  | { kind: 'city' }

const screenKinds = ['home', 'wardrobe', 'item', 'edit', 'photo', 'draftPhoto', 'add', 'recommendation', 'city']
const openScreens = 'screens'

// Телефон и Telegram перезагружают страницу когда угодно - например, пока открыт
// выбор фотографии. Без этого пользователь возвращался бы на главный экран.
function restoreScreens(): Screen[] {
  const stored = recall<Screen[]>(openScreens)
  if (stored === null || stored.length === 0 || !stored.every((screen) => screenKinds.includes(screen?.kind))) {
    return [{ kind: 'home' }]
  }
  return stored
}

// Экраны лежат стопкой: открыть - положить сверху, «Назад» - снять верхний.
// Все изменения идут через приложение, поэтому список вещей загружается один раз и дальше правится на месте.
function Screens({
  options,
  initialItems,
  initialTodayOutfit,
}: {
  options: Options
  initialItems: Item[]
  initialTodayOutfit: number[]
}) {
  const [items, setItems] = useState(initialItems)
  const [todayOutfit, setTodayOutfit] = useState(initialTodayOutfit)
  const recommendation = useRef<RecommendationMemory>({ recommendation: null, index: 0 })
  const [stack, setStack] = useState<Screen[]>(restoreScreens)
  const screen = stack[stack.length - 1]

  useEffect(() => {
    remember(openScreens, stack)
  }, [stack])

  const savedScroll = useRef<number[]>([])
  const nextScroll = useRef(0)
  useLayoutEffect(() => {
    window.scrollTo(0, nextScroll.current)
  }, [stack])

  function open(next: Screen) {
    savedScroll.current.push(window.scrollY)
    nextScroll.current = 0
    setStack((current) => [...current, next])
  }

  function back() {
    nextScroll.current = savedScroll.current.pop() ?? 0
    setStack((current) => current.slice(0, -1))
  }

  function replaceItem(updated: Item) {
    forgetRecommendation()
    setItems((current) => current.map((item) => (item.id === updated.id ? updated : item)))
  }

  function removeItems(itemIds: number[]) {
    forgetRecommendation()
    setItems((current) => current.filter((item) => !itemIds.includes(item.id)))
  }

  function openDraftPhoto(url: string, caption: string) {
    open({ kind: 'draftPhoto', url, caption })
  }

  function forgetRecommendation() {
    recommendation.current = { recommendation: null, index: 0 }
  }

  switch (screen.kind) {
    case 'home':
      return (
        <HomeScreen
          itemCount={items.length}
          todayItems={todayOutfit.flatMap((itemId) => items.filter((item) => item.id === itemId))}
          onOpenItem={(itemId) => open({ kind: 'item', itemId })}
          onOpenWardrobe={() => open({ kind: 'wardrobe' })}
          onAddItem={() => open({ kind: 'add' })}
          onRecommend={() => open({ kind: 'recommendation' })}
        />
      )
    case 'wardrobe':
      return (
        <WardrobeScreen
          options={options}
          items={items}
          wornItemIds={todayOutfit}
          onBack={back}
          onOpenItem={(itemId) => open({ kind: 'item', itemId })}
          onAddItem={() => open({ kind: 'add' })}
          onDeleted={removeItems}
        />
      )
    case 'draftPhoto':
      return <PhotoScreen url={screen.url} caption={screen.caption} onBack={back} />

    case 'add':
      return (
        <AddItemScreen
          options={options}
          onBack={back}
          onOpenPhoto={openDraftPhoto}
          onAdded={(item) => {
            forgetRecommendation()
            setItems((current) => [item, ...current])
          }}
        />
      )
    case 'recommendation':
      return (
        <RecommendationScreen
          options={options}
          memory={recommendation.current}
          wornItemIds={todayOutfit}
          onWorn={setTodayOutfit}
          onBack={back}
          onOpenItem={(itemId) => open({ kind: 'item', itemId })}
          onChooseCity={() => open({ kind: 'city' })}
          onAddItem={() => open({ kind: 'add' })}
        />
      )
    case 'city':
      return (
        <CityScreen
          onBack={back}
          onSaved={() => {
            forgetRecommendation()
            back()
          }}
        />
      )

    case 'item':
    case 'edit':
    case 'photo': {
      const item = items.find((candidate) => candidate.id === screen.itemId)
      if (item === undefined) {
        return (
          <WithBackButton onBack={back}>
            <p className="message">{errorTexts.not_found}</p>
          </WithBackButton>
        )
      }
      if (screen.kind === 'photo') {
        return <PhotoScreen url={item.photo_url} caption={item.name} onBack={back} />
      }
      if (screen.kind === 'edit') {
        return (
          <EditItemScreen
            options={options}
            item={item}
            onBack={back}
            onOpenPhoto={openDraftPhoto}
            onSaved={(updated) => {
              replaceItem(updated)
              back()
            }}
          />
        )
      }
      return (
        <ItemScreen
          options={options}
          item={item}
          onBack={back}
          onEdit={() => open({ kind: 'edit', itemId: item.id })}
          onOpenPhoto={() => open({ kind: 'photo', itemId: item.id })}
          onChanged={replaceItem}
          onDeleted={() => {
            removeItems([item.id])
            back()
          }}
        />
      )
    }
  }
}

function WithBackButton({ onBack, children }: { onBack: () => void; children: ReactNode }) {
  useBackButton(onBack)
  return children
}

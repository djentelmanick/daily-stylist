import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { fetchItems, fetchOptions, fetchTodayOutfit, reviewOutfit, wearToday, type Item, type Options } from './api'
import { CityScreen } from './CityScreen'
import { HomeScreen } from './HomeScreen'
import { AddItemScreen, EditItemScreen } from './ItemFormScreens'
import { ItemScreen, PhotoScreen } from './ItemScreen'
import { PickItemScreen } from './PickItemScreen'
import { RecommendationScreen, type RecommendationMemory } from './RecommendationScreen'
import { SettingsScreen } from './SettingsScreen'
import { recall, remember } from './session'
import { TodayOutfitScreen } from './TodayOutfitScreen'
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
  | { kind: 'settings' }
  | { kind: 'today' }
  | { kind: 'pick'; target: 'recommendation' | 'today'; outfit: number[]; replace: number | null }

const screenKinds = [
  'home',
  'wardrobe',
  'item',
  'edit',
  'photo',
  'draftPhoto',
  'add',
  'recommendation',
  'city',
  'settings',
  'today',
  'pick',
]
const openScreens = 'screens'

// Телефон и Telegram перезагружают страницу когда угодно - например, пока открыт
// выбор фотографии. Без этого пользователь возвращался бы на главный экран.
function restoreScreens(): Screen[] {
  const stored = recall<Screen[]>(openScreens)
  if (stored === null || stored.length === 0 || !stored.every((screen) => screenKinds.includes(screen?.kind))) {
    // Кнопка «Ещё варианты» в утреннем сообщении открывает приложение сразу на подборе.
    return new URLSearchParams(window.location.search).get('screen') === 'recommendation'
      ? [{ kind: 'home' }, { kind: 'recommendation' }]
      : [{ kind: 'home' }]
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

  async function saveToday(itemIds: number[]) {
    await wearToday(itemIds)
    setTodayOutfit(itemIds)
  }

  // Правка подбора ничего не записывает: образ станет образом дня по кнопке «Надеваю».
  async function replaceInRecommendation(replace: number, picked: Item) {
    const memory = recommendation.current
    const outfit = memory.recommendation?.outfits.at(memory.index)
    if (memory.recommendation === null || outfit === undefined) {
      return
    }
    const outfitItems = outfit.items.map((item) => (item.id === replace ? picked : item))
    const notes = await reviewOutfit(outfitItems.map((item) => item.id))
    memory.recommendation = {
      ...memory.recommendation,
      outfits: memory.recommendation.outfits.map((one, index) =>
        index === memory.index ? { items: outfitItems, notes } : one,
      ),
    }
  }

  async function pick(target: 'recommendation' | 'today', replace: number | null, picked: Item) {
    if (target === 'recommendation' && replace !== null) {
      await replaceInRecommendation(replace, picked)
    } else {
      const itemIds =
        replace === null
          ? [...todayOutfit, picked.id]
          : todayOutfit.map((itemId) => (itemId === replace ? picked.id : itemId))
      await saveToday(itemIds)
    }
    back()
  }

  const todayItems = todayOutfit.flatMap((itemId) => items.filter((item) => item.id === itemId))

  switch (screen.kind) {
    case 'home':
      return (
        <HomeScreen
          itemCount={items.length}
          todayItems={todayItems}
          onOpenItem={(itemId) => open({ kind: 'item', itemId })}
          onOpenToday={() => open({ kind: 'today' })}
          onOpenWardrobe={() => open({ kind: 'wardrobe' })}
          onAddItem={() => open({ kind: 'add' })}
          onRecommend={() => open({ kind: 'recommendation' })}
          onOpenSettings={() => open({ kind: 'settings' })}
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
          onReplaceItem={(replace, outfit) => open({ kind: 'pick', target: 'recommendation', outfit, replace })}
          onChooseCity={() => open({ kind: 'city' })}
          onAddItem={() => open({ kind: 'add' })}
        />
      )
    case 'today':
      return (
        <TodayOutfitScreen
          options={options}
          items={todayItems}
          onOpenItem={(itemId) => open({ kind: 'item', itemId })}
          onReplace={(replace) => open({ kind: 'pick', target: 'today', outfit: todayOutfit, replace })}
          onAdd={() => open({ kind: 'pick', target: 'today', outfit: todayOutfit, replace: null })}
          onRemove={(itemId) => saveToday(todayOutfit.filter((id) => id !== itemId))}
          onBack={back}
        />
      )
    case 'pick':
      return (
        <PickItemScreen
          options={options}
          items={items}
          outfit={screen.outfit}
          replace={screen.replace}
          onPick={(picked) => pick(screen.target, screen.replace, picked)}
          onBack={back}
        />
      )
    case 'settings':
      return <SettingsScreen onBack={back} onChooseCity={() => open({ kind: 'city' })} />
    case 'city':
      return (
        <CityScreen
          minQueryLength={options.limits.min_city_query}
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

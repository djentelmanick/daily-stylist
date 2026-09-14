import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { fetchItems, fetchOptions, type Item, type Options } from './api'
import { HomeScreen, RecommendationScreen } from './HomeScreen'
import { AddItemScreen, EditItemScreen } from './ItemFormScreens'
import { ItemScreen } from './ItemScreen'
import { getInitData, useBackButton } from './telegram'
import { errorTexts, texts } from './texts'
import { WardrobeScreen } from './WardrobeScreen'

type State =
  | { kind: 'loading' }
  | { kind: 'outsideTelegram' }
  | { kind: 'failed' }
  | { kind: 'ready'; options: Options; items: Item[] }

export function App() {
  const [state, setState] = useState<State>(() =>
    getInitData() === '' ? { kind: 'outsideTelegram' } : { kind: 'loading' },
  )

  useEffect(() => {
    if (state.kind !== 'loading') {
      return
    }

    let cancelled = false
    Promise.all([fetchOptions(), fetchItems()])
      .then(([options, items]) => {
        if (!cancelled) {
          setState({ kind: 'ready', options, items })
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
      return <p className="message">{texts.loading}</p>
    case 'outsideTelegram':
      return <p className="message">{texts.openFromBot}</p>
    case 'failed':
      return <p className="message">{texts.loadFailed}</p>
    case 'ready':
      return <Screens options={state.options} initialItems={state.items} />
  }
}

type Screen =
  | { kind: 'home' }
  | { kind: 'wardrobe' }
  | { kind: 'item'; itemId: number }
  | { kind: 'edit'; itemId: number }
  | { kind: 'add' }
  | { kind: 'recommendation' }

// Экраны лежат стопкой: открыть - положить сверху, «Назад» - снять верхний.
// Все изменения идут через приложение, поэтому список вещей загружается один раз и дальше правится на месте.
function Screens({ options, initialItems }: { options: Options; initialItems: Item[] }) {
  const [items, setItems] = useState(initialItems)
  const [stack, setStack] = useState<Screen[]>([{ kind: 'home' }])
  const screen = stack[stack.length - 1]

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
    setItems((current) => current.map((item) => (item.id === updated.id ? updated : item)))
  }

  function removeItems(itemIds: number[]) {
    setItems((current) => current.filter((item) => !itemIds.includes(item.id)))
  }

  switch (screen.kind) {
    case 'home':
      return (
        <HomeScreen
          itemCount={items.length}
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
          onBack={back}
          onOpenItem={(itemId) => open({ kind: 'item', itemId })}
          onAddItem={() => open({ kind: 'add' })}
          onDeleted={removeItems}
        />
      )
    case 'add':
      return (
        <AddItemScreen
          options={options}
          onBack={back}
          onAdded={(item) => setItems((current) => [item, ...current])}
        />
      )
    case 'recommendation':
      return (
        <WithBackButton onBack={back}>
          <RecommendationScreen />
        </WithBackButton>
      )
    case 'item':
    case 'edit': {
      const item = items.find((candidate) => candidate.id === screen.itemId)
      if (item === undefined) {
        return (
          <WithBackButton onBack={back}>
            <p className="message">{errorTexts.not_found}</p>
          </WithBackButton>
        )
      }
      if (screen.kind === 'edit') {
        return (
          <EditItemScreen
            options={options}
            item={item}
            onBack={back}
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

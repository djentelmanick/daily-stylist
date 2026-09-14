import { useRef, type MouseEvent, type PointerEvent } from 'react'

const holdMs = 500
// Палец, сдвинувшийся дальше, листает список, а не удерживает вещь.
const moveTolerancePx = 10

// Обработчики для элемента: onLongPress после удержания, onPress на обычное нажатие.
// Отпущенный после удержания палец не должен ещё и открыть вещь, поэтому click после него пропускается.
export function useLongPress(onLongPress: () => void, onPress: () => void) {
  const timer = useRef<number | undefined>(undefined)
  const start = useRef({ x: 0, y: 0 })
  const held = useRef(false)

  function cancel() {
    window.clearTimeout(timer.current)
  }

  return {
    onPointerDown(event: PointerEvent) {
      if (event.button !== 0) {
        return
      }
      held.current = false
      start.current = { x: event.clientX, y: event.clientY }
      timer.current = window.setTimeout(() => {
        held.current = true
        onLongPress()
      }, holdMs)
    },
    onPointerMove(event: PointerEvent) {
      if (Math.hypot(event.clientX - start.current.x, event.clientY - start.current.y) > moveTolerancePx) {
        cancel()
      }
    },
    onPointerUp: cancel,
    onPointerLeave: cancel,
    onPointerCancel: cancel,
    onClick() {
      if (held.current) {
        held.current = false
        return
      }
      onPress()
    },
    // Иначе Android на удержание откроет контекстное меню.
    onContextMenu(event: MouseEvent) {
      event.preventDefault()
    },
  }
}

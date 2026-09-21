import { useEffect, useRef } from 'react'

type TelegramWebApp = {
  initData: string
  colorScheme?: 'light' | 'dark'
  ready: () => void
  expand: () => void
  isVersionAtLeast: (version: string) => boolean
  showConfirm: (message: string, callback: (confirmed: boolean) => void) => void
  BackButton: {
    show: () => void
    hide: () => void
    onClick: (callback: () => void) => void
    offClick: (callback: () => void) => void
  }
  HapticFeedback: {
    impactOccurred: (style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft') => void
  }
}

declare global {
  interface Window {
    Telegram?: { WebApp?: TelegramWebApp }
  }
}

const webApp = window.Telegram?.WebApp

export function initTelegram(): void {
  webApp?.ready()
  webApp?.expand()
  // Тему Telegram отдаёт переменными, а нативные поля вроде выбора времени смотрят
  // на color-scheme: без него в тёмной теме они остаются светлыми.
  if (webApp?.colorScheme !== undefined) {
    document.documentElement.style.colorScheme = webApp.colorScheme
  }
}

export function getInitData(): string {
  return webApp?.initData ?? ''
}

// Нажатие получает только верхний обработчик: экран под открытой камерой не должен
// закрыться вместе с ней.
const backHandlers: { current: () => void }[] = []

function handleBack() {
  backHandlers[backHandlers.length - 1]?.current()
}

export function useBackButton(onBack: () => void): void {
  const latest = useRef(onBack)
  useEffect(() => {
    latest.current = onBack
  })

  useEffect(() => {
    if (!webApp?.isVersionAtLeast('6.1')) {
      return
    }
    const button = webApp.BackButton
    if (backHandlers.length === 0) {
      button.onClick(handleBack)
      button.show()
    }
    backHandlers.push(latest)
    return () => {
      backHandlers.splice(backHandlers.indexOf(latest), 1)
      if (backHandlers.length === 0) {
        button.offClick(handleBack)
        button.hide()
      }
    }
  }, [])
}

// Нативный диалог Telegram: window.confirm в части клиентов не работает.
export function confirm(message: string): Promise<boolean> {
  const app = webApp
  if (!app?.isVersionAtLeast('6.2')) {
    return Promise.resolve(window.confirm(message))
  }
  return new Promise((resolve) => app.showConfirm(message, resolve))
}

// Не light и не selectionChanged: на Android их вибрацию не почувствовать.
export function vibrate(): void {
  if (webApp?.isVersionAtLeast('6.1')) {
    webApp.HapticFeedback.impactOccurred('medium')
  }
}

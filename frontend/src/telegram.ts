import { useEffect, useRef } from 'react'

type TelegramWebApp = {
  initData: string
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
}

export function getInitData(): string {
  return webApp?.initData ?? ''
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
    const handleClick = () => latest.current()
    button.onClick(handleClick)
    button.show()
    return () => {
      button.offClick(handleClick)
      button.hide()
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

type TelegramWebApp = {
  initData: string
  ready: () => void
  expand: () => void
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

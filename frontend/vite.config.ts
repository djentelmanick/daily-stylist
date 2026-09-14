import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// Go-сервер слушает BOT_LISTEN_ADDR, по умолчанию :2000.
const backend = 'http://localhost:2000'

export default defineConfig({
  plugins: [react()],
  server: {
    // Telegram открывает приложение по адресу туннеля, а Vite по умолчанию
    // отвечает только на localhost. Это настройка только для разработки.
    allowedHosts: true,
    // Один туннель на Vite: страницу отдаёт Vite, а API и вебхук он передаёт Go.
    // '/telegram' рассчитан на путь вебхука по умолчанию - TELEGRAM_WEBHOOK_PATH.
    proxy: {
      '/api': backend,
      '/telegram': backend,
    },
  },
})

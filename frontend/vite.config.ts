import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// Go-сервер слушает BOT_LISTEN_ADDR, по умолчанию :2000. В docker адрес приходит из compose.
const backend = process.env.VITE_BACKEND_URL ?? 'http://localhost:2000'
// Хранилище фотографий из deploy/docker-compose.yml, порт S3_PORT.
const storage = process.env.VITE_STORAGE_URL ?? 'http://localhost:58333'

export default defineConfig({
  plugins: [react()],
  server: {
    // Telegram открывает приложение по адресу туннеля, а Vite по умолчанию
    // отвечает только на localhost. Это настройка только для разработки.
    allowedHosts: true,
    // Один туннель на Vite: страницу отдаёт Vite, а API и вебхук он передаёт Go.
    // '/telegram' рассчитан на путь вебхука по умолчанию - TELEGRAM_WEBHOOK_PATH.
    // '/photos' - имя бакета из S3_BUCKET: по этим ссылкам браузер ходит за
    // фотографиями в хранилище. Ни путь, ни заголовок Host менять нельзя -
    // они входят в подпись ссылки, и хранилище перестанет её принимать.
    proxy: {
      // Описание API через туннель не отдаём: локально оно открывается прямо на порту бота.
      '/api': { target: backend, bypass: (request) => (request.url?.startsWith('/api/docs') ? false : undefined) },
      '/telegram': backend,
      '/photos': storage,
    },
  },
})

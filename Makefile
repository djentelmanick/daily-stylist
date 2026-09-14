.PHONY: help tun back front check check-back check-front

help:
	@echo "Запускайте каждую команду в отдельном терминале, начиная с туннеля:"
	@echo "  make tun     - туннель ngrok на Vite (порт 5173)"
	@echo "  make back    - Go-бот и API Mini App (порт 2000)"
	@echo "  make front   - dev-сервер Vite"
	@echo ""
	@echo "Проверки:"
	@echo "  make check       - проверить всё"
	@echo "  make check-back  - тесты и линтер бэкенда"
	@echo "  make check-front - линтер, типы и сборка фронтенда"

tun:
	ngrok http 5173

back:
	cd backend && go run ./cmd/bot

front:
	cd frontend && npm run dev

check: check-back check-front

check-back:
	cd backend && go test ./... && golangci-lint run

check-front:
	cd frontend && npm run lint && npm run build

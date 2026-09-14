.PHONY: help db db-down psql migrate migrate-status migrate-down migration tun back front check check-back check-db check-front

-include backend/.env

COMPOSE = docker compose --env-file backend/.env -f deploy/docker-compose.yml

help:
	@echo "Сначала база: make db. Остальное - каждое в своём терминале, начиная с туннеля:"
	@echo "  make db      - Postgres в docker, настройки из backend/.env"
	@echo "  make db-down - остановить Postgres, данные сохранятся"
	@echo "  make psql    - консоль базы"
	@echo "  make tun     - туннель ngrok на Vite (порт 5173)"
	@echo "  make back    - Go-бот и API Mini App (порт 2000)"
	@echo "  make front   - dev-сервер Vite"
	@echo ""
	@echo "Миграции:"
	@echo "  make migrate             - накатить все новые"
	@echo "  make migrate-status      - какие применены, какие нет"
	@echo "  make migrate-down        - откатить последнюю"
	@echo "  make migration name=имя  - создать файл миграции, нужен goose CLI"
	@echo ""
	@echo "Проверки:"
	@echo "  make check       - проверить всё"
	@echo "  make check-back  - тесты и линтер бэкенда"
	@echo "  make check-db    - тесты адаптера Postgres, нужна make db"
	@echo "  make check-front - линтер, типы и сборка фронтенда"

db:
	$(COMPOSE) up -d --wait

db-down:
	$(COMPOSE) down

psql:
	$(COMPOSE) exec postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

migrate:
	cd backend && go run ./cmd/migrate up

migrate-status:
	cd backend && go run ./cmd/migrate status

migrate-down:
	cd backend && go run ./cmd/migrate down

migration:
	@test -n '$(name)' || { echo "укажите имя: make migration name=add_brand"; exit 1; }
	cd backend && goose -s create $(name) sql

tun:
	ngrok http 5173

back:
	cd backend && go run ./cmd/bot

front:
	cd frontend && npm run dev

check: check-back check-db check-front

check-back:
	cd backend && go test ./... && golangci-lint run --build-tags integration

check-db:
	@test -n '$(DATABASE_URL)' || { echo "нет DATABASE_URL: скопируйте backend/.env.example в backend/.env"; exit 1; }
	cd backend && TEST_DATABASE_URL='$(DATABASE_URL)' go test -tags integration ./internal/adapter/postgres/

check-front:
	cd frontend && npm run lint && npm run build

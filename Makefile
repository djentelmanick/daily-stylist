.PHONY: help db db-down psql migrate migrate-status migrate-down migration tun back front check check-back check-db check-s3 check-front

-include backend/.env

COMPOSE = docker compose --env-file backend/.env -f deploy/docker-compose.yml

help:
	@echo "Сначала база: make db. Остальное - каждое в своём терминале, начиная с туннеля:"
	@echo "  make db      - Postgres и хранилище фотографий в docker, настройки из backend/.env"
	@echo "  make db-down - остановить их, данные сохранятся"
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
	@echo "  make check-s3    - тесты адаптера хранилища, нужна make db"
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

check: check-back check-db check-s3 check-front

check-back:
	cd backend && go test ./... && golangci-lint run --build-tags integration

check-db:
	@test -n '$(TEST_DATABASE_URL)' || { echo "нет TEST_DATABASE_URL в backend/.env: возьмите строку из backend/.env.example"; exit 1; }
	cd backend && TEST_DATABASE_URL='$(TEST_DATABASE_URL)' go test -count=1 -tags integration ./internal/adapter/postgres/

check-s3:
	@test -n '$(TEST_S3_BUCKET)' || { echo "нет TEST_S3_BUCKET в backend/.env: возьмите строки из backend/.env.example"; exit 1; }
	cd backend && TEST_S3_ENDPOINT='$(TEST_S3_ENDPOINT)' TEST_S3_BUCKET='$(TEST_S3_BUCKET)' \
		S3_ACCESS_KEY='$(S3_ACCESS_KEY)' S3_SECRET_KEY='$(S3_SECRET_KEY)' S3_REGION='$(S3_REGION)' \
		go test -count=1 -tags integration ./internal/adapter/s3/

check-front:
	cd frontend && npm run lint && npm run build

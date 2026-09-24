.PHONY: help dev docker docker-build stop down swagger initdata db db-down vision vision-build psql migrate migrate-status migrate-down migration proto redis-cli tun back sched sender front vision-venv check check-back check-db check-s3 check-redis check-rabbit check-vision check-gigachat check-front

-include backend/.env

export TEST_DATABASE_URL TEST_REDIS_URL TEST_RABBITMQ_URL NGROK_DOMAIN
export TEST_S3_ENDPOINT TEST_S3_BUCKET S3_ACCESS_KEY S3_SECRET_KEY S3_REGION
export GIGACHAT_AUTH_KEY GIGACHAT_AUTH_URL GIGACHAT_BASE_URL GIGACHAT_SCOPE GIGACHAT_MODEL

COMPOSE = docker compose --env-file backend/.env -f deploy/docker-compose.yml
# Своя модель поднимается, только если она выбрана распознавателем в backend/.env.
VISION_PROFILE = $(if $(filter vision,$(RECOGNIZER) $(RECOGNIZER_FALLBACK)),--profile vision)
GO_MODULE = github.com/djentelmanick/daily-stylist/backend
PROTOC_PYTHON ?= $(CURDIR)/vision/.venv/bin/python

help:
	@echo "Всё сразу:"
	@echo "  make dev      - туннель и все процессы в одном терминале, инфраструктура в docker"
	@echo "  make stop     - остановить все контейнеры, оставив их на месте"
	@echo "  make down     - остановить и удалить контейнеры, данные в томах сохранятся"
	@echo "  make docker   - поднять в docker вообще всё, включая бота и фронтенд"
	@echo "  make docker-build - пересобрать образы бота и фронтенда"
	@echo ""
	@echo "По частям - каждое в своём терминале, начиная с туннеля:"
	@echo "  make db      - Postgres, Redis, RabbitMQ и хранилище фотографий в docker, настройки из backend/.env"
	@echo "  make db-down - остановить только их, данные сохранятся"
	@echo "  make vision  - своя модель распознавания, нужна при RECOGNIZER=vision"
	@echo "  make vision-build - пересобрать её образ после правок в vision/"
	@echo "  make psql    - консоль базы"
	@echo "  make redis-cli - консоль Redis"
	@echo "  make tun     - туннель ngrok на Vite (порт 5173)"
	@echo "  make back    - Go-бот и API Mini App (порт 2000)"
	@echo "  make sched   - планировщик: ставит задачи на утреннюю рассылку"
	@echo "  make sender  - отправщик: разбирает задачи и шлёт сообщения"
	@echo "  make front   - dev-сервер Vite"
	@echo ""
	@echo "Миграции:"
	@echo "  make migrate             - накатить все новые"
	@echo "  make migrate-status      - какие применены, какие нет"
	@echo "  make migrate-down        - откатить последнюю"
	@echo "  make migration name=имя  - создать файл миграции, нужен goose CLI"
	@echo ""
	@echo "Документация API:"
	@echo "  make swagger     - пересобрать описание Mini App API по аннотациям, нужен swag"
	@echo "  make initdata user=12345 - подпись Telegram для Swagger UI и curl"
	@echo ""
	@echo "Распознавание одежды:"
	@echo "  make proto       - перегенерировать код по contracts/vision.proto"
	@echo "  make vision-venv - окружение Python для тестов и линтера сервиса"
	@echo ""
	@echo "Проверки:"
	@echo "  make check       - проверить всё"
	@echo "  make check-back  - тесты и линтер бэкенда"
	@echo "  make check-db    - тесты адаптера Postgres, нужна make db"
	@echo "  make check-s3    - тесты адаптера хранилища, нужна make db"
	@echo "  make check-redis - тесты адаптера Redis, нужна make db"
	@echo "  make check-rabbit - тесты адаптера RabbitMQ, нужна make db"
	@echo "  make check-vision - тесты и линтер сервиса своей модели"
	@echo "  make check-gigachat - живой запрос к GigaChat, тратит токены; в make check не входит"
	@echo "  make check-front - линтер, типы и сборка фронтенда"

dev:
	$(COMPOSE) $(VISION_PROFILE) up -d --wait
	cd backend && go run ./cmd/migrate up
	./scripts/dev.sh

stop:
	$(COMPOSE) --profile vision --profile app stop

down:
	$(COMPOSE) --profile vision --profile app down

docker: docker-build
	$(COMPOSE) --profile app $(VISION_PROFILE) up -d --wait
	@echo "поднято. Логи: $(COMPOSE) --profile app logs -f"

docker-build:
	$(COMPOSE) --profile app build

db:
	$(COMPOSE) up -d --wait

db-down:
	$(COMPOSE) down

vision:
	$(COMPOSE) --profile vision up -d --wait vision

vision-build:
	$(COMPOSE) --profile vision build vision

psql:
	$(COMPOSE) exec postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

redis-cli:
	$(COMPOSE) exec redis redis-cli

migrate:
	cd backend && go run ./cmd/migrate up

migrate-status:
	cd backend && go run ./cmd/migrate status

migrate-down:
	cd backend && go run ./cmd/migrate down

migration:
	@test -n '$(name)' || { echo "укажите имя: make migration name=add_brand"; exit 1; }
	cd backend && goose -s create $(name) sql

initdata:
	@test -n '$(user)' || { echo "укажите пользователя: make initdata user=12345"; exit 1; }
	@cd backend && go run ./cmd/initdata -user $(user)

swagger:
	@command -v swag >/dev/null || { echo "нет swag. Поставьте его:"; \
		echo "  go install github.com/swaggo/swag/cmd/swag@v1.16.6"; exit 1; }
	cd backend && swag init -g cmd/bot/main.go -o docs --parseInternal --parseDepth 2

proto:
	@command -v protoc-gen-go >/dev/null && command -v protoc-gen-go-grpc >/dev/null || { \
		echo "нет плагинов protoc. Поставьте их:"; \
		echo "  go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.10"; \
		echo "  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1"; exit 1; }
	@command -v $(PROTOC_PYTHON) >/dev/null || { echo "нет окружения Python: make vision-venv"; exit 1; }
	cd backend && $(PROTOC_PYTHON) -m grpc_tools.protoc --proto_path=../contracts \
		--go_out=. --go_opt=module=$(GO_MODULE) \
		--go-grpc_out=. --go-grpc_opt=module=$(GO_MODULE) vision.proto
	cd vision && $(PROTOC_PYTHON) -m grpc_tools.protoc --proto_path=../contracts \
		--python_out=app/generated --pyi_out=app/generated --grpc_python_out=app/generated vision.proto
	@# Сгенерированный модуль импортирует сосед по пакету, а protoc пишет импорт верхнего уровня.
	cd vision && sed -i 's/^import vision_pb2 as/from . import vision_pb2 as/' app/generated/vision_pb2_grpc.py

vision-venv:
	cd vision && python3 -m venv .venv && .venv/bin/pip install --upgrade pip && .venv/bin/pip install -e '.[dev]'

tun:
	ngrok http 5173

back:
	cd backend && go run ./cmd/bot

sched:
	cd backend && go run ./cmd/scheduler

sender:
	cd backend && go run ./cmd/sender

front:
	cd frontend && npm run dev

check: check-back check-db check-s3 check-redis check-rabbit check-vision check-front

check-back:
	cd backend && go test ./... && golangci-lint run --build-tags integration

check-db:
	@test -n "$$TEST_DATABASE_URL" || { echo "нет TEST_DATABASE_URL в backend/.env: возьмите строку из backend/.env.example"; exit 1; }
	cd backend && go test -count=1 -tags integration ./internal/adapter/postgres/

check-s3:
	@test -n "$$TEST_S3_BUCKET" || { echo "нет TEST_S3_BUCKET в backend/.env: возьмите строки из backend/.env.example"; exit 1; }
	cd backend && go test -count=1 -tags integration ./internal/adapter/s3/

check-redis:
	@test -n "$$TEST_REDIS_URL" || { echo "нет TEST_REDIS_URL в backend/.env: возьмите строку из backend/.env.example"; exit 1; }
	cd backend && go test -count=1 -tags integration ./internal/adapter/redis/

check-rabbit:
	@test -n "$$TEST_RABBITMQ_URL" || { echo "нет TEST_RABBITMQ_URL в backend/.env: возьмите строку из backend/.env.example"; exit 1; }
	cd backend && go test -count=1 -tags integration ./internal/adapter/rabbitmq/

check-gigachat:
	@test -n "$$GIGACHAT_AUTH_KEY" || { echo "нет GIGACHAT_AUTH_KEY в backend/.env"; exit 1; }
	cd backend && go test -count=1 -tags integration ./internal/adapter/recognizers/gigachat/

check-vision:
	@test -x vision/.venv/bin/python || { echo "нет окружения Python: make vision-venv"; exit 1; }
	cd vision && .venv/bin/python -m pytest -q \
		&& .venv/bin/python -m ruff check . \
		&& .venv/bin/python -m ruff format --check .

check-front:
	cd frontend && npm run lint && npm run build

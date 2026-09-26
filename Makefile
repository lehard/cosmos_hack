# Makefile — единая точка сборки, запуска и проверок ant (AD-25).
#
# Всё выполняется в контейнерах: Go и Node на машине не нужны. Тяжёлые шаги
# (сборка, pull, compose up, проверки) берут блокировку ANT_BUILD_LOCK, чтобы
# параллельные агенты не исчерпали память машины сборки.
#
# make help — список целей.

SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c
.DEFAULT_GOAL := help
MAKEFLAGS += --no-print-directory

ROOT       := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
UID        := $(shell id -u)
GID        := $(shell id -g)

# Закреплённые образы (спайн, «Stack»).
GO_IMAGE   ?= golang:1.27.1
NODE_IMAGE ?= node:24.21.0-slim
PG_IMAGE   ?= postgres:18.6-alpine
RUNTIME_IMAGE ?= gcr.io/distroless/static-debian12:nonroot
export ANT_IMAGE   ?= ant:local
export ANT_VERSION ?= $(shell git -C $(ROOT) describe --always --dirty 2>/dev/null || echo dev)

# Кэши модулей Go, сборки, npm — на машине, общие для всех рабочих копий.
CACHE ?= $(HOME)/.cache/ant
export ANT_BUILD_LOCK ?= $(HOME)/.cache/cosmo-build.lock
$(shell mkdir -p $(CACHE)/go/gomod $(CACHE)/go/go-build $(CACHE)/go/bin $(CACHE)/go/golangci $(CACHE)/npm)

DOCKER         := $(ROOT)/deploy/scripts/docker.sh
DOCKER_LOCKED  := $(ROOT)/deploy/scripts/docker.sh --lock
COMPOSE        := $(DOCKER) compose -f $(ROOT)/compose.yaml
COMPOSE_LOCKED := $(DOCKER_LOCKED) compose -f $(ROOT)/compose.yaml

# Своя БД агента (make dev-db) подхватывается контейнером Go автоматически.
DEV_DB_ENV := $(ROOT)/.dev/db.env

GO_DOCKER_ARGS = run --rm -u $(UID):$(GID) \
	-e HOME=/tmp -e GOTOOLCHAIN=local -e GOFLAGS=-buildvcs=false \
	-e GOCACHE=/cache/go-build -e GOMODCACHE=/cache/gomod \
	-e GOLANGCI_LINT_CACHE=/cache/golangci -e TOOLBIN=/cache/bin -e ANT_CHECK_FAST=$(ANT_CHECK_FAST) \
	-v $(CACHE)/go:/cache -v $(ROOT):/src -w /src/backend \
	$(if $(wildcard $(DEV_DB_ENV)),--network ant-dev --env-file $(DEV_DB_ENV) -v $(HOME)/.config/ant-dev:/run/ant-dev:ro) \
	$(GO_EXTRA) $(GO_IMAGE)
GO_RUN = $(DOCKER_LOCKED) $(GO_DOCKER_ARGS)

NODE_RUN = $(DOCKER_LOCKED) run --rm -u $(UID):$(GID) \
	-e HOME=/tmp -e npm_config_cache=/cache/npm -e npm_config_update_notifier=false \
	-v $(CACHE):/cache -v $(ROOT):/src -w /src/frontend \
	$(NODE_IMAGE)

.PHONY: help
help: ## Список целей
	@grep -hE '^[a-zA-Z0-9_-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-18s %s\n", $$1, $$2}'

# ---------------------------------------------------------------- запуск ----

.PHONY: up down logs ps demo
up: ## Поднять систему (postgres + ant) в фоне и дождаться готовности
	$(COMPOSE_LOCKED) up -d --build --wait
	@$(MAKE) prune-dangling
	@echo "ant: http://127.0.0.1:$${ANT_HTTP_PORT:-8480}/  (healthz: /healthz, readyz: /readyz)"

down: ## Остановить систему (данные в томах сохраняются)
	$(COMPOSE) down

logs: ## Логи системы
	$(COMPOSE) logs -f --tail=200

ps: ## Состояние контейнеров системы
	$(COMPOSE) ps

demo: ## Демо-профиль: то же, что up, с ANT_PROFILE=demo
	ANT_PROFILE=demo $(MAKE) up

# ---------------------------------------------------------------- сборка ----

.PHONY: build vendor fmt run
build: ## Собрать образ ant (фронтенд встроен в бинарник)
	$(DOCKER_LOCKED) build -f $(ROOT)/deploy/Dockerfile -t $(ANT_IMAGE) \
		--build-arg VERSION=$(ANT_VERSION) \
		--build-arg GO_IMAGE=$(GO_IMAGE) --build-arg NODE_IMAGE=$(NODE_IMAGE) \
		--build-arg RUNTIME_IMAGE=$(RUNTIME_IMAGE) $(ROOT)
	@$(MAKE) prune-dangling

vendor: ## go mod tidy + go mod vendor (после изменения зависимостей)
	$(GO_RUN) sh -c 'go mod tidy && go mod vendor && cd tools && go mod tidy'

fmt: ## gofmt -w для бэкенда
	$(GO_RUN) sh -c 'gofmt -w $$(find . -name "*.go" -not -path "./vendor/*")'

ANT_DEV_HTTP_PORT ?= 8481
run: ## Запустить роль api из исходников против своей БД (make dev-db), порт ANT_DEV_HTTP_PORT
	@test -f $(DEV_DB_ENV) || { echo "сначала make dev-db"; exit 1; }
	$(DOCKER) $(GO_DOCKER_ARGS) go run ./cmd/ant -role=api -config /src/deploy/config/ant.yaml
run: GO_EXTRA = --name ant-dev-api-$(UID)-$(ANT_DEV_HTTP_PORT) -p 127.0.0.1:$(ANT_DEV_HTTP_PORT):8080 \
	-e ANT_HTTP_ADDR=:8080 -e ANT_PROFILE=$(or $(ANT_PROFILE),demo)

# --------------------------------------------------------- база агента -----

.PHONY: dev-db dev-db-drop dev-db-psql
dev-db: ## Своя БД ant_<ветка> в общем dev-postgres (127.0.0.1:55432)
	$(ROOT)/deploy/scripts/dev-db.sh create

dev-db-drop: ## Удалить свою БД ant_<ветка>
	$(ROOT)/deploy/scripts/dev-db.sh drop

dev-db-psql: ## psql в свою БД
	$(ROOT)/deploy/scripts/dev-db.sh psql

# ------------------------------------------------------------ генерация ----

.PHONY: generate
generate: ## Перегенерировать производные файлы (правила линтера слоёв; дальше — эпик 02)
	$(GO_RUN) sh -c 'bin=$$(/src/deploy/scripts/go-tools.sh ./archgen) && cd tools && $$bin/archgen -layers archgen/layers.json -out ../.golangci.yml'

# ------------------------------------------------------------- проверки ----

.PHONY: check check-backend check-frontend check-contracts check-third-party gogost-verify
check: check-backend check-frontend check-third-party check-contracts ## Все проверки: слои, детерминизм, линтеры, тесты, фронтенд, GoGOST, контракты
	@echo; echo "make check: зелёный"

check-fast: ## Быстрая проверка при слиянии пачек: как check, но тесты из кэша и без самопроверки линтеров (Д-18)
	@$(MAKE) --no-print-directory check ANT_CHECK_FAST=1

check-backend: ## Бэкенд: правила слоёв, gofmt, vet, golangci-lint, detcheck, тесты, самопроверка линтеров
	$(GO_RUN) /src/deploy/scripts/check-backend.sh

check-frontend: ## Фронтенд: ESLint (запрет ручных HTTP-вызовов), vue-tsc, сборка
	$(NODE_RUN) /src/deploy/scripts/check-frontend.sh

check-third-party: ## GoGOST: sha256, подпись автора, побайтное совпадение с архивом (офлайн)
	@if command -v zstd >/dev/null && command -v ssh-keygen >/dev/null; then \
		$(ROOT)/third_party/gogost/verify.sh; \
	else \
		echo "нет zstd или ssh-keygen — проверка GoGOST пропущена (make gogost-verify)"; \
	fi

gogost-verify: ## GoGOST полностью: + гибридная подпись (OpenSSH ≥ 10.4 в контейнере) и сверка с deckhouse/gogost v6.2.0
	$(ROOT)/third_party/gogost/verify.sh --hybrid --deckhouse

check-contracts: ## Контракты: contracts/scripts/check.sh (эпик 00), если он есть
	@if [[ -f $(ROOT)/contracts/scripts/check.sh ]]; then \
		bash $(ROOT)/contracts/scripts/check.sh; \
	else \
		echo "contracts/scripts/check.sh пока нет (эпик 00) — пропуск"; \
	fi

# ------------------------------------------------------------- лицензии ----

.PHONY: licenses
licenses: ## Перечень зависимостей с лицензиями → docs/licenses/ (NFR-SEC-2)
	mkdir -p $(ROOT)/docs/licenses
	$(GO_RUN) /src/deploy/scripts/licenses-go.sh
	$(NODE_RUN) /src/deploy/scripts/licenses-npm.sh

# -------------------------------------------- офлайн и уборка за собой ----

.PHONY: npm-cache prune-dangling clean
npm-cache: ## Наполнить frontend/.npm-cache для офлайн-сборки образа (NFR-SEC-1)
	$(NODE_RUN) sh -c 'npm ci --cache /src/frontend/.npm-cache --prefer-offline --no-audit --no-fund --ignore-scripts >/dev/null && rm -rf node_modules && echo "npm-кэш: frontend/.npm-cache"'

prune-dangling: # висящие образы ant (только с меткой ant.project=ant — чужие не трогаем)
	@$(DOCKER) image prune -f --filter "label=ant.project=ant" >/dev/null || true

clean: ## Остановить систему и убрать висящие образы ant
	-$(COMPOSE) down --remove-orphans
	@$(MAKE) prune-dangling

# ------------------------------------------ цели будущих эпиков (пока пусто) ----

.PHONY: contract-demo tamper keys load verify rebuild token-agent
contract-demo: ## Ломающее изменение контракта краснеет до отправки в 1С (эпики 02, 30)
	@echo "contract-demo: пока пусто — эпики 02, 30 (FR-111)"
tamper: ## Подделка в обход системы и её обнаружение (эпики 29, 36)
	@echo "tamper: пока пусто — эпики 29, 36 (FR-74, FR-152)"
keys: ## Ключи и генезис доверия (эпик 05)
	@echo "keys: пока пусто — эпик 05 (AD-33)"
load: ## Нагрузочный прогон 1 и N воркеров (эпик 35)
	@echo "load: пока пусто — эпик 35 (FR-107)"
verify: ## Независимый верификатор журнала (эпик 29)
	@echo "verify: пока пусто — эпик 29 (AD-9)"
rebuild: ## Пересборка проекций из журнала (эпики 07, 34)
	@echo "rebuild: пока пусто — эпики 07, 34"
token-agent: ## Агент токена и расширение браузера (эпик 38)
	@echo "token-agent: пока пусто — эпик 38 (AD-14)"

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
# Владелец ключей интерактивных демо-персон в ./.demo-keys/token-agent/ (роль init, эпик 05).
export ANT_HOST_UID ?= $(shell id -u)
export ANT_HOST_GID ?= $(shell id -g)

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

# Node для генераторов и проверок контрактов (contracts/scripts, свои закреплённые зависимости).
NODE_RUN_CONTRACTS = $(DOCKER_LOCKED) run --rm -u $(UID):$(GID) \
	-e HOME=/tmp -e npm_config_cache=/cache/npm -e npm_config_update_notifier=false \
	-v $(CACHE):/cache -v $(ROOT):/src -w /src/contracts/scripts \
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
	$(COMPOSE) --profile demo down

logs: ## Логи системы
	$(COMPOSE) logs -f --tail=200

ps: ## Состояние контейнеров системы
	$(COMPOSE) ps

demo: ## Демо-профиль: то же, что up, с ANT_PROFILE=demo и службой demo-signer (Д-59)
	ANT_PROFILE=demo COMPOSE_PROFILES=demo $(MAKE) up

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

.PHONY: generate generate-go generate-ts generate-frontend
generate: generate-go generate-ts generate-frontend ## Перегенерировать всё производное из контрактов (AD-20): Go, TS, openapi.yaml, клиент фронтенда
	@echo "make generate: готово (docs/codegen.md)"

generate-go: ## Go: правила слоёв, каталог, коды ошибок, статусы, повышатели, BPMN (структуры + XSD), типы JSON Schema, sqlc, contracts/openapi.yaml
	$(GO_RUN) /src/deploy/scripts/generate-go.sh

generate-ts: ## TS-типы из JSON Schema и каталога → frontend/src/shared/contracts (contracts/scripts/gen-ts.mjs)
	$(NODE_RUN_CONTRACTS) sh -c 'test -d node_modules && test ! package-lock.json -nt node_modules/.package-lock.json || npm ci --prefer-offline --no-audit --no-fund --loglevel=error; node gen-ts.mjs'

generate-frontend: ## Клиент orval + Vue Query, словарь статусов и коды ошибок для фронтенда (frontend/src/shared/api/generated)
	$(NODE_RUN) sh -c 'test -d node_modules || npm ci --prefer-offline --no-audit --no-fund; npm run --silent generate'

# ------------------------------------------------------------- проверки ----

.PHONY: check check-secrets check-backend check-frontend check-contracts check-third-party gogost-verify check-generated check-compat
check: check-secrets check-backend check-frontend check-third-party check-contracts check-generated check-compat ## Все проверки: секреты, слои, детерминизм, линтеры, тесты, фронтенд, GoGOST, контракты
	@echo; echo "make check: зелёный"

check-fast: ## Быстрая проверка при слиянии пачек: как check, но тесты из кэша и без самопроверки линтеров (Д-18)
	@$(MAKE) --no-print-directory check ANT_CHECK_FAST=1

check-secrets: ## Секреты: в репозитории нет закрытых ключей и токенов (NFR-SEC-1; исключения — deploy/scripts/check-secrets.allow)
	@bash $(ROOT)/deploy/scripts/check-secrets.sh

check-backend: ## Бэкенд: правила слоёв, gofmt, vet, golangci-lint, detcheck, тесты, самопроверка линтеров
	$(GO_RUN) /src/deploy/scripts/check-backend.sh

check-frontend: ## Фронтенд: ESLint (запрет ручных HTTP-вызовов), столы ↔ реестр, клиент не устарел, vue-tsc, тесты, сборка
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

# Базовая версия контракта для обнаружения ломающих изменений (AD-20, О8): прошлый
# тег contract-v* или ветка main (на main — предыдущий коммит). Переопределение — ANT_CONTRACT_BASE.
CONTRACT_BASE ?= $(or $(ANT_CONTRACT_BASE),$(shell git -C $(ROOT) describe --tags --abbrev=0 --match 'contract-v*' 2>/dev/null),$(shell b=main; [ "$$(git -C $(ROOT) rev-parse HEAD 2>/dev/null)" = "$$(git -C $(ROOT) rev-parse $$b 2>/dev/null)" ] && b=HEAD~1; echo $$b))

# Всё, что пишет make generate (AD-20: руками не правится, конфликт — перегенерацией).
GENERATED_PATHS = backend/.golangci.yml backend/internal/contracts contracts/openapi.yaml contracts/bpmn-ext/ant.xsd \
	docs/bpmn-ext-properties.md frontend/src/shared/contracts frontend/src/shared/api/generated

check-generated: ## Сгенерированное не устарело: make generate + git diff --exit-code (AD-20)
	@$(MAKE) --no-print-directory generate
	@cd $(ROOT) && gen="$(GENERATED_PATHS)"; if [ -n "$$(git status --porcelain --untracked-files=all -- $$gen)" ]; then \
		echo "Сгенерированное расходится с закоммиченным — выполните make generate и закоммитьте:"; git status --short -- $$gen; exit 1; fi
	@echo "check-generated: сгенерированное совпадает с источниками"

# Уровни правил oasdiff — contracts/oasdiff-levels.txt: новое значение перечисления
# в ответе — совместимое изменение (AD-20, FR-29, случай «неизвестное значение
# перечисления»: клиент показывает UNKNOWN(значение)), как в compat-rules.mjs для событий.
check-compat: ## Ломающие изменения контракта: oasdiff breaking (openapi.yaml) и @asyncapi/diff (asyncapi.yaml со схемами) против $(CONTRACT_BASE)
	@rm -rf $(ROOT)/.dev/compat && mkdir -p $(ROOT)/.dev/compat/base
	@git -C $(ROOT) archive $(CONTRACT_BASE) contracts/events contracts/openapi.yaml 2>/dev/null | tar -x -C $(ROOT)/.dev/compat/base 2>/dev/null || true
	@echo "check-compat: база $(CONTRACT_BASE)"
	$(GO_RUN) sh -c 'bin=$$(/src/deploy/scripts/go-tools.sh github.com/oasdiff/oasdiff); b=/src/.dev/compat/base/contracts/openapi.yaml; if [ -s $$b ]; then $$bin/oasdiff breaking $$b /src/contracts/openapi.yaml --severity-levels /src/contracts/oasdiff-levels.txt --err-ignore /src/contracts/oasdiff-err-ignore.txt --fail-on ERR && echo "oasdiff: ломающих изменений HTTP API нет"; else echo "oasdiff: базовой openapi.yaml нет — пропуск"; fi'
	$(NODE_RUN_CONTRACTS) sh -c 'test -d node_modules && test ! package-lock.json -nt node_modules/.package-lock.json || npm ci --prefer-offline --no-audit --no-fund --loglevel=error; node check-compat.mjs /src/.dev/compat/base/contracts/events/asyncapi.yaml'

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

# ------------------------------------------------------------ симуляция ----

SIM_PKGS = ./internal/domain/simulation/... ./internal/application/simulation/... \
	./internal/infrastructure/storage/simulation/... ./internal/infrastructure/transport/simulation/... \
	./internal/infrastructure/fixtures/simulation/...

.PHONY: sim-check sim-streams
sim-check: ## Симуляция без движка (эпик 32): генератор по seed, потоки JSONL по схемам контракта, ожидания, автосверка всех сценариев на заготовках приёма
	$(GO_RUN) sh -c 'go test -count=1 $(SIM_PKGS) && go test -count=1 -run TestAutocheckOnFakes -v ./internal/infrastructure/storage/simulation/ | grep -E "табло|итого|строк"'

sim-streams: ## Пересобрать потоки прогонов scenarios/definitions/streams из определений (после правки scenarios/definitions)
	$(GO_RUN) sh -c 'ANT_UPDATE_STREAMS=1 go test -count=1 -run TestStreams ./internal/infrastructure/storage/simulation/'

# ------------------------------------------ цели будущих эпиков (пока пусто) ----

.PHONY: contract-demo tamper keys load verify rebuild token-agent
contract-demo: ## Ломающее изменение контракта краснеет до отправки в 1С (FR-111; эпик 30 — отправка в stand 1С)
	$(NODE_RUN_CONTRACTS) sh -c 'test -d node_modules && test ! package-lock.json -nt node_modules/.package-lock.json || npm ci --prefer-offline --no-audit --no-fund --loglevel=error; node contract-demo.mjs'
# make tamper [ATTACK=1|2|3|all] — три атаки AD-28 демо-инструментом (профили
# fixtures и demo), попытка того же через API и независимая проверка следом.
tamper: ## Подделка в обход системы и её обнаружение: три атаки (AD-28), попытка через API, верификатор (эпик 29)
	$(COMPOSE) --profile tools run --rm --no-deps tamper -attack $(or $(ATTACK),all) -api http://ant:8080
	@echo; echo "Независимая проверка (верификатор, отчёт — хранителю):"
	@$(COMPOSE) run --rm --no-deps verifier -once || test $$? -eq 3
	@echo; echo "Индикатор целостности на столах загорится, когда ant заберёт отчёт у хранителя (security.interval)."
keys: ## Ключи и генезис доверия (эпик 05): ant init (повтор ничего не меняет); ключи интерактивных демо-персон — в ./.demo-keys/token-agent/
	@mkdir -p $(ROOT)/.demo-keys/token-agent
	$(COMPOSE_LOCKED) run --rm init
# Не запускать на общей машине разработки (Д-75) — только на выделенном стенде.
# make load [N=4] [SCENARIO=MS-1] [SEED=…] [OUTAGE=20] — два раздельных прогона
# одного seed (1 и N воркеров, посреди второго — падение и подъём копии воркера),
# отчёты и сравнение хешей в scenarios/load/out/. Образ ant:load собирается из
# рабочей копии, если его нет (LOAD_BUILD=1 — пересобрать); демо-стенд не трогается:
# свой проект compose ant-load и порты 8484/8494.
LOAD_IMAGE ?= ant:load
load: ## Нагрузочный прогон 1 и N воркеров: rebuild_hash и state_hash «1 = N», задержки, потери, повторы (эпик 35)
	@if [[ "$(LOAD_BUILD)" == 1 ]] || ! $(DOCKER) image inspect $(LOAD_IMAGE) >/dev/null 2>&1; then \
		$(MAKE) build ANT_IMAGE=$(LOAD_IMAGE); fi
	ANT_IMAGE=$(LOAD_IMAGE) $(ROOT)/scenarios/load/load.sh
normative-sync: ## Встроенные копии normative/ во всех seed/ и во входе мира заготовок + перегенерация заготовок (после любой правки normative/)
	@$(ROOT)/deploy/scripts/sync-normative.sh
	$(GO_RUN) go test ./internal/infrastructure/fixtures/world -update
verify: ## Независимый верификатор журнала: проверка по запросу, подписанный отчёт — хранителю (эпик 29, AD-9)
	@$(COMPOSE) run --rm --no-deps verifier -once || test $$? -eq 3
rebuild: ## Пересборка проекций из журнала (ant rebuild; ITEM=‹item_id› — одно изделие: повтор после «обработка остановлена»)
	@# Полная пересборка — при остановленных worker и projector (служба ant),
	@# после неё служба запускается снова; -item — на работающей системе.
	@if [[ -z "$(ITEM)" ]]; then $(COMPOSE) stop ant; fi; \
	status=0; $(COMPOSE_LOCKED) run --rm --no-deps ant -role=rebuild $(if $(ITEM),-item=$(ITEM)) || status=$$?; \
	if [[ -z "$(ITEM)" ]]; then $(COMPOSE) start ant; fi; exit $$status
# make token-agent — пакет рабочего места под macOS arm64 (эпик 38, AD-14, Д-72):
# расширение «Главный — подпись» (распакованное) с пакетом подписи WASM, агент
# токена token-agent (darwin/arm64; по возможности darwin/amd64 и linux/amd64),
# скрипт установки и README. Перед сборкой — Go-тесты ядра и сверка «отпечаток
# WASM = отпечаток Go» в Node. Демо-ключи персон кладутся в пакет, если есть
# DEMO_KEYS (по умолчанию ./.demo-keys/token-agent). Итог — dist/glavny-token-agent-macos-arm64.tar.gz.
TA_DIST     := $(ROOT)/dist/glavny-token-agent
DEMO_KEYS   ?= $(ROOT)/.demo-keys/token-agent
TA_EXT_FILES := manifest.json background.js content.js common.js confirm.html confirm.js manage.html manage.js popup.html popup.js style.css icon128.png
token-agent: ## Пакет рабочего места: расширение + WASM + token-agent (macOS arm64) + установка (эпик 38)
	@rm -rf $(TA_DIST) && mkdir -p $(TA_DIST)/extension $(TA_DIST)/bin
	$(GO_RUN) bash /src/backend/cmd/token-agent/build.sh $(ANT_VERSION) /src/dist/glavny-token-agent
	@cd $(ROOT)/extension && cp $(TA_EXT_FILES) $(TA_DIST)/extension/
	$(DOCKER_LOCKED) run --rm -u $(UID):$(GID) -v $(ROOT):/src -w /src $(NODE_IMAGE) node extension/test/wasm-vectors.mjs dist/glavny-token-agent/extension
	@cp $(ROOT)/extension/install.sh $(ROOT)/extension/README.md $(TA_DIST)/
	@if [ -d "$(DEMO_KEYS)" ] && ls $(DEMO_KEYS)/*-ta@1.key.json >/dev/null 2>&1; then \
		mkdir -p $(TA_DIST)/demo-keys && cp $(DEMO_KEYS)/*.key.json $(TA_DIST)/demo-keys/ && chmod 600 $(TA_DIST)/demo-keys/*; \
		echo "демо-ключи персон: $(DEMO_KEYS) → demo-keys/ (секреты демо-стенда, не для промышленной эксплуатации)"; \
	else echo "демо-ключей нет в $(DEMO_KEYS) — make keys или DEMO_KEYS=‹каталог›"; fi
	@cd $(ROOT)/dist && tar -czf glavny-token-agent-macos-arm64.tar.gz glavny-token-agent
	@echo "пакет: $(ROOT)/dist/glavny-token-agent-macos-arm64.tar.gz (распакованный — $(TA_DIST))"

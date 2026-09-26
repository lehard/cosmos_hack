# Кодогенерация и проверка контрактов

Материалы кейса §6.2 одним файлом (AD-20, FR-110, FR-111, NFR-DEV-1, NFR-DEV-2; критерии О7, О8). Здесь:

- источники контрактов;
- версии генераторов;
- команды;
- перечень генерируемых компонентов;
- проверки рассинхронизации;
- пример изменения контракта v1 → v2.

Черновик эпика 02.

## Принцип

У каждого вида контракта один источник. Из него генерируется код. Сгенерированное лежит в репозитории и руками не правится: источник записан в шапке каждого файла. Конфликт в сгенерированных файлах решается перегенерацией, а не ручным слиянием. Цепочка генерации идёт без циклов:

```text
contracts/events/*.json, catalog.yaml, errors.yaml, statuses.yaml, constants.yaml, upcasters/, bpmn-ext/ant.json
        │  make generate-go (contractgen, go-jsonschema)
        ▼
backend/internal/contracts/**  (Go-типы — чистые данные, доступны домену)
        │  go build; операции Huma (infrastructure/transport/‹модуль›)
        ▼
contracts/openapi.yaml  (ant -openapi)
        │  make generate-frontend (orval)
        ▼
frontend/src/shared/api/generated/**  (клиент orval + Vue Query)

contracts/**/*.json ──make generate-ts (json-schema-to-typescript)──▶ frontend/src/shared/contracts/**
```

## Источники

| Вид контракта | Источник | Решение |
|---|---|---|
| События: конверт, общие определения, данные каждого типа | `contracts/events/common/*.json`, `contracts/events/‹семейство›/‹тип›.v‹N›.json` | AD-20, AD-40 |
| Каталог типов записей: эмитент, вид, поток, ось, класс, критичность | `contracts/events/catalog.yaml` | AD-40 |
| AsyncAPI 3.0 и SSE-канал | `contracts/events/asyncapi.yaml` (собирается из каталога `contracts/scripts/asyncapi.mjs`) | AD-20, AD-21 |
| Повышатели версий | `contracts/events/upcasters/*.yaml` (декларативная таблица) | AD-20, FR-110 |
| Запись журнала | `contracts/journal/entry.schema.json` | AD-44 |
| Криптоформаты: DSSE, контрольная точка | `contracts/crypto/*.schema.json` | AD-10, AD-8 |
| Собственные процессы: агент токена, верификатор, телеметрия stand-ов | `contracts/internal/**` | AD-46 |
| Коды ошибок RFC 9457 | `contracts/errors.yaml`, `contracts/problem.schema.json` | FR-28 |
| Словарь статусов | `contracts/statuses.yaml` | AD-30 |
| Константы | `contracts/constants.yaml` | AD-3, AD-10 |
| Расширение BPMN | `contracts/bpmn-ext/ant.json` (дескриптор moddle), `rules.yaml` | AD-17 |
| Затравка нормативного слоя | `contracts/normative/*.schema.json` | AD-17, AD-31 |
| HTTP API | операции Huma в Go (`backend/internal/infrastructure/transport/‹модуль›`) | AD-20, AD-40 |
| Правила слоёв | `backend/tools/archgen/layers.json` | AD-1 |
| SQL модулей | `backend/internal/infrastructure/storage/‹модуль›/sqlc.yaml` + запросы | AD-1 |

## Версии генераторов

Версии Go-инструментов закреплены директивами `tool` в `backend/tools/go.mod`. Версии npm-инструментов — точными значениями в `package.json` и в lock-файле.

| Инструмент | Версия | Где закреплён | Что делает |
|---|---|---|---|
| Go | 1.27.1 | образ `golang:1.27.1` | сборка и генерация |
| Huma | v2.39.1 | `backend/go.mod` | операции HTTP → OpenAPI 3.1 |
| go-jsonschema | v0.24.1 | `backend/tools/go.mod` | JSON Schema → Go-типы |
| sqlc | v1.31.1 | `backend/tools/go.mod` | SQL → Go |
| oasdiff | v1.32.1 | `backend/tools/go.mod` | ломающие изменения HTTP API |
| golangci-lint (depguard, forbidigo) | v2.14.0 | `backend/tools/go.mod` | правила слоёв и детерминизма |
| contractgen | исходник в репозитории | `backend/tools/contractgen` | каталог, коды, статусы, константы, повышатели, BPMN (Go + XSD + таблица), копия схем |
| archgen | исходник в репозитории | `backend/tools/archgen` | `.golangci.yml` из `layers.json` |
| emitcheck, detcheck | исходник в репозитории | `backend/tools/{emitcheck,detcheck}` | «модуль эмитит чужой тип»; float и обход map в домене |
| Node.js | 24.21.0 | образ `node:24.21.0-slim` | генерация и проверки на JS |
| json-schema-to-typescript | 16.0.0 | `contracts/scripts/package.json` | JSON Schema → TS-типы |
| @asyncapi/parser, @asyncapi/diff | 3.6.3, 0.5.0 | `contracts/scripts/package.json` | проверка и сравнение AsyncAPI |
| orval | 8.37.0 | `frontend/package.json` | OpenAPI → клиент + Vue Query |

## Команды

Все команды выполняются в контейнерах: Go и Node на машине не нужны.

```sh
make generate           # всё: generate-go → generate-ts → generate-frontend
make generate-go        # Go: слои, contractgen, go-jsonschema, sqlc, contracts/openapi.yaml
make generate-ts        # TS-типы контрактов → frontend/src/shared/contracts
make generate-frontend  # клиент orval + Vue Query, статусы, коды ошибок → frontend/src/shared/api/generated
make check-generated    # перегенерация + сверка с закоммиченным (git status по сгенерированным путям)
make check-compat       # ломающие изменения: oasdiff breaking + @asyncapi/diff против прошлого тега contract-v* или main
make contract-demo      # воспроизводимое несовместимое изменение краснеет до отправки в 1С (FR-111)
```

## Что генерируется

| Результат | Генератор | Источник |
|---|---|---|
| `backend/.golangci.yml` | archgen | `layers.json` |
| `backend/internal/contracts/catalog/catalog_gen.go` | contractgen | `catalog.yaml` |
| `backend/internal/contracts/errcodes/errcodes_gen.go` | contractgen | `errors.yaml` |
| `backend/internal/contracts/statuses/statuses_gen.go` | contractgen | `statuses.yaml` |
| `backend/internal/contracts/constants/constants_gen.go` | contractgen | `constants.yaml` |
| `backend/internal/contracts/upcast/upcast_gen.go` — повышатели, чистые функции | contractgen | `upcasters/*.yaml` |
| `backend/internal/contracts/events/registry_gen.go` — «тип + версия → Go-тип data» | contractgen | схемы событий |
| `backend/internal/contracts/bpmnext/bpmnext_gen.go`, `contracts/bpmn-ext/ant.xsd`, `docs/bpmn-ext-properties.md` | contractgen | `bpmn-ext/ant.json` |
| `backend/internal/contracts/schemas/**` — копия схем для встраивания в бинарник: приём проверяет тела теми же схемами | contractgen | `contracts/**/*.json` |
| `backend/internal/contracts/{events,journal,crypto,procs,normative,problem}/*_gen.go` | go-jsonschema | JSON Schema |
| `backend/internal/infrastructure/storage/‹модуль›/**` (запросы) | sqlc | SQL модуля |
| `contracts/openapi.yaml` | Huma (`ant -openapi`) | операции Go |
| `frontend/src/shared/contracts/{events,journal,crypto,procs,constants,catalog}.ts` | json-schema-to-typescript, `gen-ts.mjs` | JSON Schema, YAML |
| `frontend/src/shared/api/generated/**` | orval, `frontend/scripts/generate.mjs` | `openapi.yaml`, `statuses.yaml`, `errors.yaml`, `asyncapi.yaml` |

## Соглашения операций HTTP API

- **Идентификатор и x-ant-action.** Каждая операция объявляет `x-ant-action`:
  - `id` — он же `operationId`, ключ Casbin и действие `@casl`, вида `‹модуль›.‹объект›.‹действие›`;
  - `class` — `read | record | protective | permissive | irreversible`;
  - `critical` и `ca_group`;
  - `owner`;
  - `subject`;
  - `guards`;
  - `emits` — типы записей, которые пишет операция;
  - `signature_level`.

  Описание проверяет `httpapi.Register`: при нарушении выгрузка спецификации падает. Файл проверяет `contracts/scripts/check-openapi.mjs`.
- **Чтение.** Операции чтения принимают `axis` (`occurred` — «как было», `recorded` — «что мы знали») и `as_of` (AD-21, AD-22).
- **Команды.** Тело команды несёт `command_id` (UUIDv7), `basis_seq`, `policy_seq`, `workplace_id` и необязательный подписанный пакет `signature` (AD-7, AD-39, AD-14). Ответ — квитанция: `seq`, `event_ids`, `ca_ref`.
- **Режим данных.** Режим `fixtures | live` передаётся в заголовке ответа `Ant-Backend` (AD-36).
- **Ошибки.** Ошибки — `application/problem+json` с `code` из `errors.yaml`. Пока операция не реализована, она отвечает 501 `api.not_implemented`.
- **Живые обновления.** SSE — операция `journal.stream.subscribe`. Сообщение `EntityChanged` вынесено в `components` (AD-21).

## Проверки рассинхронизации (make check)

| Проверка | Что ловит |
|---|---|
| `check-generated` | сгенерированное устарело относительно источников |
| `check-compat` (oasdiff breaking, @asyncapi/diff + правила FR-29 в `compat-rules.mjs`) | несовместимое изменение HTTP API или событий без новой мажорной версии |
| `check-openapi.mjs` + самопроверка | у операции нет `x-ant-action` или класса; модуль эмитит чужой тип; действие политики не соответствует операции |
| `emitcheck` + самопроверка в `arch-selftest.sh` | доменный код строит запись чужого модуля (`kernel.NewReaction` / `NewAddressed`) |
| `kernel.NewReaction` / `NewAddressed` (во время работы) | тип записи не из каталога или чужого модуля |
| `archgen -check`, depguard, forbidigo, detcheck | нарушение направления слоёв и детерминизма домена |

## Пример изменения контракта v1 → v2

`equipment.state.changed` (FR-110):

1. **v1** (`contracts/events/equipment/equipment.state.changed.v1.json`) — одно поле `machine_state`. **v2** (`…v2.json`) разделяет его на `execution`, `condition` и `controller_mode`. Изменение несовместимое, поэтому это новая мажорная версия.
2. Повышатель — декларативная таблица `contracts/events/upcasters/equipment.state.changed.v1-to-v2.yaml`.
3. `make generate` превращает таблицу в чистую функцию `upcast.EquipmentStateChangedV1ToV2` (`backend/internal/contracts/upcast/upcast_gen.go`). Её применяют при чтении воркер, воспроизведение и верификатор. Журнал хранит исходный конверт v1.
4. Примеры — `contracts/events/examples/versions/`.

**Та же правка без новой версии** — например, удалить обязательное поле из v1: `make contract-demo` показывает, что `make check-compat` краснеет до отправки результата в 1С.

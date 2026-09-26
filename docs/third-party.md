# Собственный и заимствованный код, лицензии, режимные требования

Что в репозитории написано командой, что заимствовано, под какими лицензиями и как это согласуется с требованиями закрытого контура, приказа ФСТЭК № 117 и Указа № 166. Опоры: NFR-SEC-1, NFR-SEC-2, PRD §6.3 (перечень собственного и заимствованного кода в сдаточных материалах), соглашение спайна «Заимствованный код», AD-1, AD-25.

**Полный машинный перечень** всех зависимостей с лицензиями — `docs/licenses/go.csv` и `docs/licenses/npm.csv`, их обновляет `make licenses` (`go-licenses` + `license-checker-rseidelsohn`). Этот документ — человеческое пояснение к нему; при расхождении прав перечень.


**Лицензия проекта.** «Главный» распространяется на условиях GNU GPL v3 — файл [`LICENSE`](../LICENSE) в корне; выбор согласован со встроенной библиотекой GoGOST (GPLv3). Иллюстрации камер (TIG Aluminium 5083) — CC BY-SA 4.0, см. [`images/vision-demo/ATTRIBUTION.md`](images/vision-demo/ATTRIBUTION.md).

## 1. Что написано командой

Весь код в `backend/internal/**` (домен, сценарии приложения, адаптеры, stand-ы внешних систем), точки входа `backend/cmd/*` (`ant`, `keeper`, `verifier`, `token-agent`, `edge-agent`, `demo-signer`, `tamper`), фронтенд `frontend/src/**`, расширение браузера `extension/`, контракты `contracts/**`, нормативный слой `normative/**`, сценарии `scenarios/**`, сборка и развёртывание `deploy/**`, `Makefile`, документация `docs/**`. Генераторы и линтеры проекта — `backend/tools/**`.

Сгенерированный код (`*_gen.go`, `frontend/src/shared/api/generated/`) — производный от наших спецификаций; генераторы — заимствованные инструменты (раздел 3).

## 2. Заимствованные библиотеки, входящие в систему

Версии — из спайна (раздел Stack) и файлов зависимостей; лицензии — по данным проектов, проверены при выборе стека и повторно проверяются `make licenses`.

### 2.1. Бэкенд (Go)

| Библиотека | Версия | Лицензия | Зачем |
|---|---|---|---|
| Go, стандартная библиотека (`crypto/mldsa`, `encoding/json/jsontext`, `crypto/tls`) | 1.27.1 | BSD-3-Clause | язык; ML-DSA-65, канонизация JSON (RFC 8785), TLS 1.3 |
| Huma v2 | v2.39.1 | MIT | описание операций API в Go и выгрузка OpenAPI 3.1, SSE |
| pgx v5 | v5.11.0 | MIT | драйвер PostgreSQL |
| goose | v3.28.0 | MIT | миграции |
| Casbin | v3.10.0 | Apache-2.0 | вычисление решений о доступе |
| alexedwards/scs v2, scs/pgxstore | v2.9.0 / v0.0.0-20251002162104-209de6e426de | MIT | сеансы веба, хранение сеансов в PostgreSQL |
| golang.org/x/crypto (argon2id) | v0.57.0 | BSD-3-Clause | хеширование паролей |
| golang.org/x/time/rate | v0.16.0 | BSD-3-Clause | ограничение частоты |
| **GoGOST** | 7.0.0 | **GPLv3** | ГОСТ Р 34.11-2012 (Стрибог-256), ГОСТ Р 34.10-2012; поставляется исходниками в `third_party/gogost`, происхождение и sha256 — `third_party/gogost/SOURCE` |
| santhosh-tekuri/jsonschema | v6.0.3 | Apache-2.0 | проверка тел событий схемами при приёме |
| prometheus/client_golang | v1.24.1 | Apache-2.0 | метрики `/metrics` |
| makiuchi-d/gozxing | v0.1.1 | MIT | чтение QR со сканов |
| go.yaml.in/yaml/v3 | v3.0.5 | MIT / Apache-2.0 | чтение YAML: конфигурация, нормативный слой |

Конверт DSSE (`backend/internal/domain/signing/envelope.go`) и кодировщик QR (`backend/internal/application/documents/qr.go`) — собственный код, без библиотек.

### 2.2. Фронтенд и расширение

| Библиотека | Версия | Лицензия | Зачем |
|---|---|---|---|
| Vue | 3.5.43 | MIT | интерфейс |
| Vue Router | 5.3.1 | MIT | маршрутизация |
| Pinia | 4.0.3 | MIT | состояние интерфейса и сеанса |
| Naive UI | 2.45.3 | MIT | компоненты |
| @tanstack/vue-query | 5.103.2 | MIT | кэш серверных данных |
| @casl/ability, @casl/vue | 7.0.1 / 3.0.1 | MIT | отображение допустимых действий |
| **bpmn-js** | 18.30.1 | MIT **с условием**: водяной знак bpmn.io удалять нельзя | живая карта и редактор процесса; водяной знак виден (AD-21) |
| vue-i18n | 11.4.2 | MIT | тексты интерфейса |
| @fontsource/pt-sans, @fontsource/pt-mono | 5.3.0 | OFL-1.1 | шрифты PT Sans и PT Mono |
| @vicons/tabler | 0.13.0 | MIT | иконки |

Шрифты и иконки — локальные файлы в сборке; внешних CDN нет (NFR-SEC-1). Панель свойств редактора процесса (`frontend/src/features/process-editor/ui/PropertiesPanel.vue`) и графики (контрольная карта — SVG, `frontend/src/widgets/control-chart/ui/ControlChartView.vue`) — собственный код, без библиотек.

## 3. Инструменты сборки (в систему не входят)

| Инструмент | Версия | Лицензия | Зачем |
|---|---|---|---|
| sqlc | v1.31.1 | MIT | генерация кода доступа к БД |
| go-jsonschema | v0.24.1 | MIT | Go-типы из JSON Schema |
| json-schema-to-typescript | 16.0.0 | MIT | TS-типы из JSON Schema |
| orval | 8.37.0 | MIT | клиент фронтенда из OpenAPI |
| @asyncapi/diff, @asyncapi/parser | 0.5.0 / 3.6.3 | Apache-2.0 | проверка и сравнение AsyncAPI |
| ajv, ajv-formats, @readme/openapi-parser, jsdom, yaml | 8.20.0 / 3.0.1 / 9.0.0 / 26.1.0 / 2.9.1 | MIT (yaml — ISC) | проверки контрактов (`contracts/scripts`) |
| oasdiff | v1.32.1 | Apache-2.0 | поиск ломающих изменений OpenAPI |
| golangci-lint (depguard, forbidigo) | v2.14.0 | GPL-3.0 | линтеры, правила слоёв |
| google/go-licenses | v2.0.1 | Apache-2.0 | перечень лицензий Go |
| license-checker-rseidelsohn | 5.0.1 | BSD-3-Clause | перечень лицензий npm |
| TypeScript, Vite, ESLint | 5.9.3 / 8.3.1 / 10.11.0 | Apache-2.0 / MIT / MIT | сборка фронтенда |
| Vitest, happy-dom, vue-tsc, @vue/test-utils | 5.0.2 / 20.14.5 / 3.3.11 / 2.5.1 | MIT | тесты и проверка типов фронтенда |
| Node.js (образ `node:24.21.0-slim`), образ `golang:1.27.1`, образ исполнения distroless static | — | MIT / BSD-3-Clause / Apache-2.0 | только стадии сборки и исполнения в контейнере |

Версии генераторов закреплены в репозитории (отдельный `backend/tools/go.mod`, точные версии и lock-файл npm); подробности и команды — [codegen.md](codegen.md).

## 4. Инфраструктура

| Компонент | Версия | Лицензия | Замечание |
|---|---|---|---|
| PostgreSQL (образ `postgres:18.6-alpine`) | 18.6 | PostgreSQL License | журнал, проекции, аренды |
| Docker Compose | ≥ v2.29.7 | Apache-2.0 | запуск одной командой |

## 5. Замечания по лицензиям

- **GoGOST — GPLv3.** Библиотека встраивается в бинарник `ant`. В закрытом контуре без передачи бинарника третьим лицам вопрос распространения не возникает; при передаче системы другой организации это учитывается (исходники GoGOST поставляются в `third_party/gogost` вместе с `COPYING`). В промышленной эксплуатации криптография за портами `Signer` / `Verifier` / `Cipher` заменяется сертифицированным СКЗИ — зависимость от GoGOST в подписях и шифровании уходит; хеш цепочки Стрибог-256 остаётся за тем же пакетом или СКЗИ по решению заказчика.
- **bpmn-js** — MIT с требованием сохранить водяной знак bpmn.io; водяной знак виден на карте и в редакторе.
- **Картинки камер `docs/images/vision-demo/`** — кадры набора TIG Aluminium 5083 (D. Bacioiu и др., University of Birmingham), лицензия **CC BY-SA 4.0**: иллюстрации сигналов КТ-3, не снимки изделия; источник, изменения и sha256 — [ATTRIBUTION.md](images/vision-demo/ATTRIBUTION.md).
- **golangci-lint — GPL-3.0**, но это инструмент сборки: в систему не входит.
- Все прочие зависимости — разрешительные лицензии (MIT, BSD, ISC, Apache-2.0, PostgreSQL); шрифты PT — OFL-1.1. Лицензий AGPL и коммерческих в составе нет; варианты с ними (например, отдельные серверы идентичности и коммерческие UI-библиотеки) отвергнуты при выборе стека.

## 6. Режимные требования

| Требование | Как выполняется |
|---|---|
| Закрытый контур (NFR-SEC-1): без интернета во время работы | Go-зависимости — `backend/vendor`; npm — офлайн-кэш (`make npm-cache`); базовые образы загружаются заранее; шрифты и иконки встроены; внешних сервисов во время работы нет |
| Указ № 166: иностранное ПО на значимых объектах КИИ | ядро — на открытом стеке, собирается из исходников, без иностранных сервисов и облачных зависимостей во время работы; движок BPMN — собственный (Camunda / Flowable отвергнуты); используемые библиотеки иностранного происхождения (bpmn-js и др.) — открытый код, поставляемый в сборке, **оговорены здесь**; окончательное решение о допустимости состава для конкретного объекта — за заказчиком |
| Приказ ФСТЭК № 117 (с 01.03.2026) | соответствие групп мер механизмам системы — [threat-model.md](threat-model.md) |
| Криптография | в MVP GoGOST и `crypto/mldsa` — не СКЗИ; в промышленной эксплуатации — сертифицированное СКЗИ класса не ниже КС2 за теми же портами (AD-32) |
| Происхождение заимствованного кода | GoGOST — архив автора с проверкой подписи и сверкой с зеркалом (`make gogost-verify`, `third_party/gogost/SOURCE`); остальное — точные версии в `go.sum` и lock-файле npm |
| Внешняя CV-модель и датасет (кейс §7.2) | собственная CV-модель не обучается, внешняя не подключается; иллюстрации к сигналам — открытый набор TIG Aluminium 5083 (Kaggle, CC BY-SA 4.0) с пометкой «ИЛЛЮСТРАЦИЯ», источником, лицензией и автором (PRD §11.12, [vision-camera-project.md](vision-camera-project.md)) |

## Сверено с кодом

- Таблицы разделов 2–3 сверены с `backend/go.mod`, `backend/vendor/modules.txt` (33 модуля), `backend/tools/go.mod`, `frontend/package.json` и `contracts/scripts/package.json`: версии совпадают. Из спайна в код не попали go-securesystemslib, yeqown/go-qrcode, ECharts / vue-echarts, bpmn-js-properties-panel и @asyncapi/cli — их заменил собственный код (разделы 2.1, 2.2) или @asyncapi/parser и @asyncapi/diff (`contracts/scripts/check.sh`). Строки таблиц исправлены, добавлено пришедшее с кодом: scs/pgxstore, go.yaml.in/yaml/v3, vue-i18n, шрифты PT, @vicons/tabler.
- `docs/licenses/npm.csv` (137 пакетов) соответствует `frontend/package.json`. `docs/licenses/go.csv` отстал от кода: в нём 8 пакетов, а `golang.org/x/sync` и `golang.org/x/text` указаны версий v0.17.0 и v0.29.0 при v0.23.0 и v0.42.0 в `backend/vendor/modules.txt`. Перечень пересобирается `make licenses`; до пересборки Go-часть сверена по `backend/vendor`.
- Нестандартные лицензии среди транзитивных зависимостей. Go (`backend/vendor`): только MIT, BSD-3-Clause и Apache-2.0, кроме GoGOST (GPL-3.0); у go.yaml.in/yaml/v3 две лицензии, MIT и Apache-2.0. npm (`docs/licenses/npm.csv`): у bpmn-js `Custom: LICENSE` (MIT с водяным знаком, раздел 5), у шрифтов PT — OFL-1.1, у lightningcss — MPL-2.0, остальное MIT, ISC, BSD и Apache-2.0. lightningcss, vite, rolldown, esbuild и typescript попадают в перечень как необязательные peer-зависимости vue, vue-router и pinia (`frontend/package-lock.json`, пометка `devOptional`). Это инструменты сборки, в бандл интерфейса они не входят. Лицензий AGPL и коммерческих нет.
- Инструменты, добавленные после каркаса. Go-инструменты закреплены директивой `tool` в `backend/tools/go.mod`: sqlc, go-jsonschema, golangci-lint, oasdiff, go-licenses, goose (версии и лицензии — раздел 3). Собственные генераторы и линтеры — `backend/tools/{archgen,contractgen,detcheck,emitcheck}`. npm-инструменты: Vitest, happy-dom, vue-tsc, @vue/test-utils, eslint-plugin-vue и typescript-eslint (MIT, `frontend/package.json`), а также ajv, ajv-formats, @readme/openapi-parser, jsdom и json-schema-to-typescript (MIT) и yaml (ISC) в `contracts/scripts/package.json`. Лицензии взяты из lock-файлов.
- Набор иллюстраций описан в `normative/vision/illustrations.v1.yaml` (`id: tig-al5083`): «TIG Aluminium 5083», https://www.kaggle.com/datasets/danielbacioiu/tig-aluminium-5083, автор Daniel Bacioiu, CC BY-SA 4.0, ссылка на статью Bacioiu и др. (Journal of Manufacturing Processes, 2019), пометка «ИЛЛЮСТРАЦИЯ». Там же сопоставлены пять классов набора с кодами дефектов. Файлов набора в репозитории нет: их подключают на краю (`edge-agent -illustrations ‹каталог›`). Если файла или хранилища нет, иллюстрация не прикладывается, а в ограничениях наблюдения пишется «ИЛЛЮСТРАЦИЯ не приложена (…)» с набором, классом, ссылкой, лицензией и автором (`backend/internal/domain/vision/illustration.go`). Лицензию манифест требует сверить на странице набора перед сдачей материалов.

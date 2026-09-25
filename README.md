# cosmo-controller — система `ant`

Доверенная система контроля качества деталей — решение кейса КосмоХакатона 2026
«Интеллектуальный контроль качества деталей». В коде система называется `ant`.

Требования — [docs/prd.md](docs/prd.md), архитектура — [docs/architecture-spine.md](docs/architecture-spine.md).

## Запуск одной командой

Нужен только Docker (Engine ≥ 24, Compose ≥ 2.29.7). Go и Node.js на машине не нужны:
всё собирается в контейнерах.

```sh
docker compose up
```

Первый запуск собирает образ (несколько минут), дальше — секунды. После старта:

| Адрес | Что |
|---|---|
| http://127.0.0.1:8480/ | интерфейс |
| http://127.0.0.1:8480/healthz | процесс жив |
| http://127.0.0.1:8480/readyz | самопроверка пройдена, БД отвечает |

Порт и адрес меняются переменными: `ANT_HTTP_PORT=9000 ANT_HTTP_BIND=0.0.0.0 docker compose up`.
Профиль конфигурации — `ANT_PROFILE=fixtures | demo | load | prod` (по умолчанию `demo`).

Остановить — `docker compose down`; удалить и данные — `docker compose down -v`.

### Что поднимается

| Служба | Что делает |
|---|---|
| `secrets-init` | один раз создаёт пароль БД в томе `secrets` (секретов в репозитории нет) |
| `postgres` | PostgreSQL 18.6 |
| `migrate` | разовая роль `ant -role=migrate`: схемы и миграции модулей |
| `ant` | роль `api`: HTTP API и встроенный интерфейс |

Лимиты памяти рассчитаны на ~2,4 ГБ на всю систему.

### Закрытый контур

Во время работы система не обращается в интернет: шрифты и иконки встроены, Go-зависимости
лежат в `backend/vendor`. Чтобы собрать образ без сети, заранее на машине с сетью:

```sh
make npm-cache                     # npm-пакеты → frontend/.npm-cache
docker pull golang:1.27.1 node:24.21.0-slim postgres:18.6-alpine gcr.io/distroless/static-debian12:nonroot
```

## Разработка

Все цели — `make help`. Главные:

| Цель | Что делает |
|---|---|
| `make up` / `make down` | поднять систему в фоне и дождаться готовности / остановить |
| `make build` | собрать образ `ant:local` |
| `make check` | все проверки: направление зависимостей, детерминизм домена, линтеры, тесты, фронтенд, контракты |
| `make generate` | перегенерировать производные файлы |
| `make dev-db` | своя БД `ant_<ветка>` в общем dev-postgres (для параллельной работы агентов) |
| `make run` | роль `api` из исходников против своей БД |
| `make licenses` | перечень зависимостей с лицензиями → `docs/licenses/` |

### Что проверяет `make check`

- **Слои (AD-1, NFR-ARCH-1, NFR-DEV-3).** `golangci-lint` + `depguard`: домен не импортирует
  приложение и инфраструктуру; доменные модули импортируют только модули раньше себя в порядке
  `reference → signing → access → item → process → vision → quality → machinelogs → documents →
  nonconformity → analysis → notifications → crossitem → engine`; зоны `storage`, `transport`,
  `integration`, `fixtures` не импортируют друг друга. Правила генерируются из
  `backend/tools/archgen/layers.json` в `backend/.golangci.yml`.
- **Детерминизм домена (AD-4, NFR-DET-1).** `forbidigo`: в `domain/**` нет `time.Now` и других
  часов, `math/rand`, `crypto/rand`, пакета `os`; анализатор `detcheck`: нет `float32/float64` и
  обхода `map` без сортировки.
- **Самопроверка.** В копии кода создаются пробные нарушения (импорт инфраструктуры из домена,
  `time.Now()` в домене, float и др.) — каждое должно быть поймано, иначе проверка красная.
- **Фронтенд.** ESLint запрещает ручные HTTP-вызовы (`fetch`, `axios` и т. п.) вне
  `src/shared/api/sse` — только сгенерированный клиент (AD-20); `vue-tsc`; сборка.
- **Контракты.** `contracts/scripts/check.sh`, если он есть.

### Структура

```text
backend/          Go-модуль ant: cmd/*, internal/{domain,application,infrastructure}/…
frontend/         Vue 3 + Vite + TypeScript + Naive UI, слои FSD
contracts/        единственные источники контрактов (AD-20)
normative/        нормативный слой: BPMN, шаблоны, политика
scenarios/        тестовые сценарии и ожидаемые результаты
extension/        браузерное расширение агента токена
deploy/           Dockerfile, compose, config/ant.yaml, k8s, служебные скрипты
third_party/      заимствованный код исходниками (GoGOST)
docs/             документы для жюри
```

### Конфигурация

`deploy/config/ant.yaml`: секция `defaults`, поверх неё — `profiles.<профиль>`, поверх —
переменные `ANT_*` (путь ключа через `_`: `http.addr` → `ANT_HTTP_ADDR`). Секретов в переменных
нет: пароль БД и ключи — только файлами (`ANT_DB_PASSWORD_FILE`).

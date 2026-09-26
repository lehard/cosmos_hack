# Наблюдаемость и брокер сообщений: как подключить OpenTelemetry и Kafka

Ответ на вопрос кейса §3.1: «как переданная заказчику система может быть впоследствии дополнена экспортом метрик через OpenTelemetry или подключением брокера сообщений, например Kafka? Необходимо указать затрагиваемые модули, интерфейсы, настройки и проверки». Опоры: AD-6, AD-7, AD-18, AD-35, AD-45; FR-41, FR-113, FR-114; критерий О6.

**Статус.** В MVP построены структурированные логи и метрики Prometheus (`/metrics`); адаптеры OpenTelemetry и Kafka — описание (PRD §6.2, раздел «Отложено» спайна). Порты, в которые они встают, и их ключи конфигурации есть в коде: `ports.adapters.telemetry`, `ports.adapters.work_feed`, `ports.adapters.publisher` в `deploy/config/ant.yaml`. Значение `otlp` ключа `telemetry` сейчас отвергается при старте с сообщением «адаптер otlp описан (FR-113), в MVP не собран»; секций `telemetry.otlp` и `transport.kafka` в конфигурации нет — они появятся вместе с адаптером (ниже помечены «добавит адаптер»).

## 1. Принцип

Внешняя зависимость — только за портом; у каждого порта ключ конфигурации и общий контрактный тест, который проходят все адаптеры (AD-35). Поэтому подключение OpenTelemetry или Kafka — это **новый адаптер + ключ конфигурации + прохождение уже существующего контрактного теста**. Домен, сценарии приложения, схемы событий и API не меняются.

**Kafka — транспорт, а не источник истины.** Журнал остаётся в своём хранилище (`JournalStore`), с цепочками, контрольными точками и верификатором. Kafka может нести работу воркерам, исходящие сообщения и межзаводской обмен, но не заменяет журнал.

## 2. Что есть в MVP

| Механизм | Где | Что даёт |
|---|---|---|
| Логи | `backend/internal/infrastructure/observability/logging` (`slog`, JSON) | поля `event_id`, `correlation_id`, `item_id`, `run_id`, `module`; без содержимого записей и без секретов |
| Метрики | порт `Telemetry` (`backend/internal/application/platform/ports.go`), адаптер Prometheus (`backend/internal/infrastructure/observability/telemetry/prometheus.go`); `GET /metrics` роли api (`backend/cmd/ant/api.go`, `backend/cmd/ant/ops.go`) | приём: `ant_ingest_messages_total`, `ant_ingest_latency_seconds`, `ant_ingest_delivery_delay_seconds`, `ant_ingest_quarantine_open`, `ant_ingest_source_completeness_bp` (FR-41), `ant_ingest_signature_unverified_total`; движок: `ant_fold_seconds`, `ant_event_to_sse_seconds` (бюджет FR-2), `ant_processing_failed_total`; эксплуатация: `ant_ops_component_up`, `ant_ops_queue_pending`, `ant_ops_consumer_lag_seq`, `ant_ops_outbox_quarantined`, `ant_ops_integration_degraded`, `ant_ops_quarantine_open`, `ant_ops_stopped_items` |
| Проверки живости | `/healthz`, `/readyz` | процесс жив; самопроверка после старта пройдена, БД отвечает |
| Состояние компонентов | модуль `ops`, стол администратора | сервисы, очереди, карантин, интеграции, остановленные изделия, последний отчёт верификатора (FR-127) |
| Транспорт работы | порт `WorkFeed` (`ports.adapters.work_feed: postgres`), адаптер «партиции в Postgres» | партиции по `hash(item_id)`, аренды с эпохой, `LISTEN/NOTIFY` как сигнал «есть новое» |
| Исходящие | порт `Publisher` (`ports.adapters.publisher: http`), адаптер HTTP с повтором | бизнес-ключ идемпотентности, повтор только при транспортных ошибках |
| Шина безопасности | семейство `security` журнала + подписчики | экспорт во внешний мониторинг ИБ — JSON-строки в файл или syslog (AD-24) |

Счётчики дублей, отказов, задержки и объём карантина — операционные метрики, а не проекции журнала (AD-7).

## 3. OpenTelemetry

### 3.1. Что затрагивается

| Слой | Модуль / пакет | Что меняется |
|---|---|---|
| infrastructure/observability | новый пакет `backend/internal/infrastructure/observability/otel` | адаптер порта `Telemetry` на OTLP (метрики; по желанию — трассы и логи) |
| cmd | `backend/cmd/ant` (сборка зависимостей), так же `keeper`, `verifier`, `edge-agent` | выбор адаптера по ключу конфигурации |
| конфигурация | `deploy/config/ant.yaml` | значение `otlp` ключа `ports.adapters.telemetry` + секция `telemetry.otlp` (добавит адаптер) |
| deploy | `deploy/compose`, `deploy/k8s` | коллектор OpenTelemetry внутри контура (отдельный контейнер) |
| **не меняются** | `domain/*`, `application/*`, `contracts/*`, фронтенд | — |

Домен не знает о телеметрии: порт `Telemetry` вызывают только `application` и инфраструктура. Метрики и их имена определены один раз; адаптер Prometheus и адаптер OTLP экспортируют один и тот же набор.

### 3.2. Интерфейс

Порт `Telemetry` объявлен в `backend/internal/application/platform/ports.go`:

```go
type Telemetry interface {
	Counter(name string, delta int64, labels ...string)
	Observe(name string, value time.Duration, labels ...string)
	Gauge(name string, value int64, labels ...string)
}
```

Адаптеры в MVP — `prometheus` (по умолчанию), `memory` (для тестов), `nop`; выбор — `telemetry.New` по ключу `ports.adapters.telemetry` (`backend/internal/infrastructure/observability/telemetry/select.go`). Адаптер OTLP реализует тот же интерфейс; трассы — расширение порта (начало и конец спана с атрибутами `correlation_id`, `event_id`, `item_id`, `run_id`). Трассы связываются по `correlation_id` / `causation_id` конверта события, поэтому путь «источник → приём → журнал → свёртка → outbox → 1С» виден одной трассой даже через асинхронные границы.

### 3.3. Настройки

Сейчас в `deploy/config/ant.yaml`:

```yaml
defaults:
  ports:
    adapters:
      telemetry: prometheus   # prometheus | memory | nop; otlp — отвергается до появления адаптера
```

Добавит адаптер OTLP (предлагаемые ключи):

```yaml
defaults:
  ports:
    adapters:
      telemetry: otlp           # или prometheus
telemetry:
  otlp:
    endpoint: otel-collector:4317   # коллектор в закрытом контуре
    protocol: grpc                  # grpc | http
    tls_ca_file: /run/ant-secrets/otel/ca.pem
    interval: 15s
    resource:
      service.namespace: ant
      deployment.environment: prod
  traces:
    enabled: false
    sample_ratio_bp: 1000           # 10 %, без float (AD-4 для единообразия)
```

Переменные — по общему правилу: `ANT_PORTS_ADAPTERS_TELEMETRY`, `ANT_TELEMETRY_OTLP_ENDPOINT` и т. д. Секретов в переменных нет: сертификаты — файлами в томах.

### 3.4. Проверки

- **Контрактный тест порта `Telemetry`** — `TestTelemetryContract` в `backend/internal/infrastructure/observability/telemetry/telemetry_test.go` (уже проходят `prometheus` и `memory`): одна и та же последовательность вызовов даёт одинаковый набор метрик; адаптер OTLP добавляется в тот же тест.
- **Смоук-проверка в compose**: коллектор получает метрики `ant_event_to_sse_seconds` и счётчик дублей после прогона короткого сценария.
- **Закрытый контур**: экспорт идёт только на коллектор внутри контура; при недоступности коллектора система работает, метрики теряются, а не блокируют обработку (буфер экспортёра ограничен).
- Ограничение: в логи и атрибуты спанов не попадают содержимое записей и секреты (то же правило, что для логов).

## 4. Kafka

### 4.1. Где Kafka может встать

| Порт | Сейчас | С Kafka | Что сохраняется |
|---|---|---|---|
| `WorkFeed` (работа воркерам) | партиции в Postgres, аренды, `LISTEN/NOTIFY` | топик с ключом `item_id`, группа потребителей; число партиций топика = P | порядок в пределах изделия; один обработчик на изделие; курсор потребителя пишется в той же транзакции, что и выход (AD-45) |
| `Publisher` (исходящие во внешние системы) | HTTP с повтором | топик на систему; адаптер-мост к 1С / MES на стороне получателя | бизнес-ключ идемпотентности; повтор только при транспортных ошибках; квитанция — событием в журнал |
| Порт межзаводского обмена | HTTP с досылкой | брокер между предприятиями (FR-131) | проверка подписей получателем; квитанции событиями |
| Подписчики шины безопасности | файл / syslog | топик для внешней системы мониторинга ИБ | новый подписчик — без изменения источников (AD-24) |

Журнал в Kafka **не переносится**: основная цепочка, контрольные точки, верификатор и `journal.Append` остаются в `JournalStore` (Postgres). Порядок записи задаёт журнал, а Kafka получает из него уже записанные события.

### 4.2. Что затрагивается

| Слой | Модуль / пакет | Что меняется |
|---|---|---|
| infrastructure/transport | новые адаптеры `backend/internal/infrastructure/transport/journal/kafka` (для `WorkFeed`) и `…/transport/‹модуль›/kafka` (для `Publisher`) | реализация портов |
| infrastructure/integration | мосты stand-ов или реальных систем, если получатель читает из Kafka | новый адаптер получателя |
| infrastructure/storage/journal/feed | источник для публикации в Kafka — тот же курсор по `seq` (transactional outbox над журналом) | ничего нового в журнале |
| cmd | `backend/cmd/ant` | выбор адаптера по ключу конфигурации |
| конфигурация, deploy | `deploy/config/ant.yaml`, `deploy/compose`, `deploy/k8s` | значения `kafka` / `broker` ключей `ports.adapters.work_feed` и `ports.adapters.publisher` + секция `transport.kafka` (добавит адаптер), брокер в контуре |
| **не меняются** | `domain/*`, `application/*`, `contracts/events`, `contracts/openapi.yaml`, фронтенд | — |

### 4.3. Настройки

Сейчас в `deploy/config/ant.yaml`:

```yaml
defaults:
  ports:
    adapters:
      work_feed: postgres     # (kafka — описание)
      publisher: http         # (broker — описание)
  engine:
    partitions: 16            # P; в профиле load — 64
```

Добавит адаптер Kafka (предлагаемые ключи):

```yaml
defaults:
  ports:
    adapters:
      work_feed: kafka
      publisher: broker
transport:
  kafka:
    brokers: ["kafka-1:9093", "kafka-2:9093"]
    work_topic: ant.work         # партиций топика = engine.partitions (P)
    consumer_group: ant-worker
    out_topics:
      onec: ant.out.erp.1c
    tls_ca_file: /run/ant-secrets/kafka/ca.pem
    client_cert_file: /run/ant-secrets/kafka/client.pem
    client_key_file: /run/ant-secrets/kafka/client.key
```

### 4.4. Правила, которые адаптер обязан соблюсти

- Ключ сообщения — `item_id` (внутренний ID, AD-41); число партиций топика равно P; изменение P — перераспределение, как у аренд в Postgres.
- Доставка «не меньше одного раза» достаточна: приём идемпотентен по `source_id + event_id`, свёртка идемпотентна, вклады изделий заменяются целиком (AD-7, AD-45). Транзакции Kafka «ровно один раз» не требуются.
- Курсор потребителя, выход и эпоха аренды — в одной транзакции Postgres; смещение Kafka подтверждается после фиксации транзакции. Упавший воркер — перебалансировка группы вместо истечения аренды.
- При воспроизведении и прогоне «по новым правилам» адаптер `Publisher` не запускается (роль `outbox` не поднята).

### 4.5. Проверки

- **Контрактный тест `WorkFeed`** — тот же, что для адаптера Postgres: порядок в пределах изделия, один обработчик на изделие, отсутствие потерь при падении воркера.
- **`state_hash` на Kafka**: раздельные прогоны одного seed на Postgres-`WorkFeed` и на Kafka-`WorkFeed` дают один и тот же итоговый хеш состояния ([scaling.md](scaling.md)).
- **Эталонные сообщения** для `Publisher`: адаптер Kafka публикует те же сообщения, что HTTP, с тем же бизнес-ключом.
- **Сценарий недоступности брокера**: обработка останавливается и догоняет после восстановления; дублей в показателях нет.

## 5. Режимные замечания

Apache Kafka и OpenTelemetry — открытое ПО; используются внутри закрытого контура без обращения к внешним сервисам (NFR-SEC-1, NFR-SEC-2). Выбор дистрибутива брокера и коллектора для значимого объекта КИИ — решение заказчика по его требованиям (Указ № 166); архитектура от этого не зависит, потому что адаптеры стоят за портами.

## Сверено с кодом

- Порты и ключи — `backend/internal/application/platform/ports.go` (`PortTelemetry`, `PortWorkFeed`, `PortPublisher`), значения — `deploy/config/ant.yaml`, секция `ports.adapters`.
- Адаптеры телеметрии и выбор по ключу — `backend/internal/infrastructure/observability/telemetry/`; контрактный тест — `TestTelemetryContract`, обработчик `/metrics` — `TestPrometheusHandler`, отказ `otlp` — `TestNewByKey`.
- Пакетов `observability/otel` и `transport/‹…›/kafka` нет: это место будущих адаптеров.

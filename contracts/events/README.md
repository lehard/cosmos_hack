# События: конверт, каталог, схемы

Контракт событий v1 (кейс §4.3–4.7, FR-27…29, FR-110; AD-2, AD-20, AD-40, AD-44).

- `common/envelope.v1.json` — конверт, общий для всех типов записей.
- `common/defs.v1.json` — общие определения полей (время, идентификаторы, измерения, дефект, материалы, оси статусов…).
- `catalog.yaml` — каталог типов записей; его формат — `catalog.schema.json`.
- `‹семейство›/‹тип›.v‹N›.json` — схема поля `data` для типа и версии.
- `asyncapi.yaml` — AsyncAPI 3.0: каналы семейств, сообщения всех типов, SSE-канал живых обновлений; собирается из каталога (`scripts/asyncapi.mjs`).
- `examples/` — примеры сообщений: пять случаев изменения контракта (FR-29) и версии v1 → v2 (FR-110).

## Конверт

Событие — JSON-объект: поля конверта сверху, содержимое типа — в `data`. Подписывается целиком (DSSE над каноническим JSON RFC 8785, AD-10); в журнал попадает исходный подписанный конверт без изменений (AD-20).

| Поле | Обяз. | Смысл |
|---|---|---|
| `event_id` | да | UUIDv7 у фактов и команд (у подписанной команды — это её `command_id`); UUIDv5 у реакций, производных и событий сценария (AD-38, AD-44) |
| `event_type` | да | тип из каталога `семейство.сущность.действие` |
| `schema_version` | да | мажорная версия схемы `data` |
| `source_id` | да | источник: устройство, шлюз, терминал, агент токена, `ant:‹модуль›` для записей ядра, `partner:‹код›`, `‹run_id›/‹источник›` в прогоне |
| `source_seq` | для устройств, шлюзов, агентов | номер у источника: непрерывность проверяет верификатор (AD-7, AD-9) |
| `source_kind`, `reliability` | для фактов | вид источника и надёжность (AD-2, FR-140) |
| `occurred_at` | да | время возникновения; история строится по нему (FR-27) |
| `correlation_id` | да | сквозная цепочка (например, один сигнал → решение → учёт в 1С) |
| `causation_id` | да (null у корневого) | непосредственная причина |
| `run_id` | в прогоне | пространство имён прогона сценария (AD-38) |
| `item_id` | если известен | внутренний ID изделия (записи ядра, терминалы, получившие ID от системы) |
| `item_ref` | если есть носитель | носитель, по которому приём разрешает `item_id` (AD-16, AD-41) |
| `corrects` | для исправлений | исправление ранее записанной записи новой записью (FR-122) |
| `reaction` | для реакций | правило, режим, слот, версия, `supersedes`, причины, `basis_seq` (AD-3) |
| `command` | для решений | `command_id`, `basis_seq`, проверенные потоки, `policy_seq`, уровень подписи, рабочее место (AD-39, AD-14) |
| `integrity` | да | метаданные целостности под подписью: версия формата, криптопрофиль, подписанты (AD-10, кейс §6.3) |
| `data` | да | содержимое типа |

`received_at` и `recorded_at` ставит только ядро — это поля записи журнала (`contracts/journal/entry.schema.json`), а не конверта (FR-27). Порядок в пределах изделия — `occurred_at` → `received_at` → `event_id` (AD-5).

### Соответствие полям кейса §4.4

Кейс допускает другую схему при документировании соответствий. Наши имена:

| Кейс §4.4 / FR-27 | У нас | Где |
|---|---|---|
| `event_id`, `event_type`, `schema_version` | те же | конверт |
| `occurred_at`, `source_id` | те же | конверт |
| `item_id` | `item_ref` (носитель у источника) → `item_id` (внутренний, после разрешения носителя) | конверт; в журнале — `item_id` и `carrier_ref` (AD-16, AD-41) |
| `item_type_id` | `item_type_id` | `data` (`item.item.registered`, `reference.item_type.defined`) |
| `line_id`, `station_id` | `line_id`, `station_id` | `data` операций и контроля |
| `operation_run_id` | `operation_run_id` | `data`; привязку фактов оборудования к выполнению делает только межизделийная стадия (AD-29) |
| `operator_id`, `equipment_id` | `operator_id` (псевдоним или null — «неизвестно»), `equipment_id` | `data` |
| `operation_started_at`, `operation_finished_at` | `operation_started_at`, `operation_finished_at` | `data` операций |
| `reported_duration` | `reported_duration` (значение, единица, смысл интервала, происхождение) | `data` `operation.run.finished` |
| `inspection_result` | `outcome`: `defect_indicated` / `no_defect_indicated` / `unable_to_assess` | `data` `inspection.result.recorded` |
| `defects` | `defects[]` (вид, описание, компонент, зона, место, тяжесть, размер, измерение и допуск) | там же |
| `confidence`, `observation_quality` | `analyzer_confidence_bp`, `observation_quality_bp` — раздельно, в базисных пунктах | там же |
| `action_type` | отдельные типы семейства `operator` | `operator.*` |
| `machine_state` | v1: `machine_state`; v2: `execution` + `condition` + `controller_mode` | `equipment.state.changed` |
| `evidence_refs` | `evidence_refs[]` — адреса материалов | `data` |
| `analyzer_version` | `versions.analyzer_version` — в векторе версий (AD-29) | `data` `inspection.result.recorded` |
| FR-27: `inspection_method`, `processing_state`, `recipe_ref`, `limitations[]`, `identification_level` | `method`, `processing_state`, `versions.recipe_ref`, `limitations[]`, `item_ref.identification_level` | конверт и `data` |
| FR-27: метаданные целостности | `integrity` | конверт |

## Каталог

Поля типа в `catalog.yaml`:

| Поле | Значения | Решение |
|---|---|---|
| `emitter` | единственный модуль-эмитент (= владелец семейства, кроме `emitter_exceptions`) | AD-40 |
| `role` | роль процесса, где исполняется функция-эмитент | AD-40 |
| `kind` | `fact` — источник; `reaction` — вывод движка; `decision` — решение человека; `service` — время, политика, ключи, контрольные точки, генезис, карантин | AD-2 |
| `stream` | вид потока (`item`, `lot`, `incident`, … `global`) | AD-39 |
| `axis` | ось §3b, которую запись меняет; только у модуля-владельца оси | AD-30 |
| `action_class` | `protective` / `permissive` / `irreversible`; `record` — не действие | AD-27 |
| `critical`, `ca_group` | критическое действие и его группа | AD-28 |
| `guard_relevant` | меняет версию потока для гардов | AD-39 |
| `publish_stage` | запись изделия идёт на межизделийную стадию | AD-42 |
| `provenance` | допустимые классы происхождения подписи | AD-2 |
| `versions`, `current_version` | мажорные версии схемы `data` | AD-20 |

Правила, общие для всех типов:

- Факт обязан нести `source_kind` и `reliability`; реакция — блок `reaction`; решение — блок `command`.
- Запись с `corrects` — критическое действие группы `protected_data` независимо от типа (FR-122, AD-28).
- Разрешающее действие не может опираться только на факты `server-attested` (AD-2); справочники, от которых зависят права и предусловия, не меняются фактами `server-attested`.

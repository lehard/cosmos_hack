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
| `source_seq` | для устройств, шлюзов, агентов (схема не требует) | номер у источника: непрерывность проверяет верификатор (AD-7, AD-9); без номера разрывы не проверяются |
| `source_kind`, `reliability` | для фактов (схема не требует) | вид источника и надёжность (AD-2, FR-140) |
| `occurred_at` | да | время возникновения; история строится по нему (FR-27) |
| `correlation_id` | да | сквозная цепочка (например, один сигнал → решение → учёт в 1С) |
| `causation_id` | да (null у корневого) | непосредственная причина |
| `run_id` | в прогоне | пространство имён прогона сценария (AD-38) |
| `item_id` | если известен (схема не требует) | внутренний ID изделия (записи ядра, терминалы, получившие ID от системы) |
| `item_ref` | если есть носитель (схема не требует) | носитель, по которому приём разрешает `item_id` (AD-16, AD-41) |
| `corrects` | для исправлений | исправление ранее записанной записи новой записью (FR-122) |
| `reaction` | для реакций | правило, режим, слот, версия, `supersedes`, причины, `basis_seq` (AD-3) |
| `command` | для решений | `command_id`, `basis_seq`, проверенные потоки, `policy_seq`, уровень подписи, рабочее место (AD-39, AD-14) |
| `integrity` | да | метаданные целостности под подписью: версия формата, криптопрофиль, подписанты (AD-10, кейс §6.3) |
| `data` | да | содержимое типа |

`received_at` и `recorded_at` ставит только ядро — это поля записи журнала (`contracts/journal/entry.schema.json`), а не конверта (FR-27); `received_at`, присланный источником, игнорируется (п. 1 под таблицей соответствия). Порядок в пределах изделия — `occurred_at` → `received_at` → `event_id` (AD-5).

### Соответствие полям кейса §4.4

Кейс допускает другую схему при документировании соответствий. Наши имена; «обяз.» — по `required` схем, остальное необязательно. Сводка обязательных полей по типам — [ниже](#обязательные-поля-по-типам); та же таблица с пояснениями — `docs/data-model.md` §3.1.

| Кейс §4.4 / FR-27 | У нас | Где |
|---|---|---|
| `event_id`, `event_type`, `schema_version` | те же | конверт, обяз. |
| `occurred_at`, `source_id` | те же | конверт, обяз. |
| (время поступления) | `received_at` | запись журнала, не конверт: ставит приём (п. 1 под таблицей) |
| `item_id` | `item_ref` (носитель у источника) → `item_id` (внутренний, после разрешения носителя) | конверт, оба необяз.; в журнале — `item_id` и `carrier_ref` (AD-16, AD-41); без изделия — поток `global` ([ниже](#обязательные-поля-по-типам)) |
| `item_type_id` | `item_type_id` | в событиях операций и контроля не передаётся: обяз. в `data` `item.item.registered`, справочник — `reference.item_type.defined`; к операциям и результатам контроля присоединяется по `item_id` (п. 2) |
| `line_id`, `station_id` | `line_id`, `station_id` | только `data` `operation.run.started` (необяз.); `station_id` — ещё в фактах `equipment.*`. В `inspection.result.recorded` их нет — выводятся через выполнение операции (п. 3) |
| `operation_run_id` | `operation_run_id` | обяз. в `operation.run.*`; необяз. в `inspection.result.recorded`, `operator.*`, `operation.precondition.failed`; привязку фактов оборудования к выполнению делает только межизделийная стадия (`equipment.event.bound`, AD-29) |
| `operator_id` | `operator_id`; в `inspection.result.recorded` — **`inspector_id`** | `operation.run.started` — обяз., псевдоним или `null` («неизвестно», FR-123); `operator.*` — обяз., только псевдоним; в `operator.action.observed` поля нет (есть `workplace_id`); в результате контроля исполнитель — контролёр `inspector_id` (необяз., псевдоним или `null`) |
| `equipment_id` | `equipment_id` | обяз. в `equipment.*` и `operator.mode.changed`; необяз. в `operation.run.started` и `inspection.result.recorded` (средство контроля) |
| `operation_started_at`, `operation_finished_at` | те же | `data` `operation.run.started` (начало), `operation.run.finished` (оба); необяз. |
| `reported_duration` | `reported_duration` {`value`, `unit`, `meaning`, `origin`} — внутри все четыре обяз. | `data` `operation.run.finished`, необяз.; `meaning`: `active_processing` / `time_at_station` / `other` (+ `meaning_note`); `origin`: `source_reported` / `system_computed`. Итог выполнения — `completion`: `completed` / `interrupted` (обяз.) |
| `inspection_result` | `outcome`: `defect_indicated` / `no_defect_indicated` / `unable_to_assess` (+ `unable_reason`) | `data` `inspection.result.recorded`, обяз. вместе с `method`, `phase`, `processing_state` |
| `defects` | `defects[]`: обяз. `zone_id`, `severity` (есть `unknown`); необяз. `defect_type_code`, `description`, `component_ref`, `location`, `size`, `measured`, `tolerance`, `stage_confidence_bp` | там же; массив необяз. |
| `confidence`, `observation_quality` | `analyzer_confidence_bp`, `observation_quality_bp` — раздельно, в базисных пунктах; уверенность ступени — `stages[].confidence_bp`, признака — `defects[].stage_confidence_bp` | `inspection.result.recorded`, `operator.action.observed`; все необяз. |
| `action_type` | отдельные типы семейства `operator` | `operator.step.confirmed`, `.check.skipped`, `.override.performed`, `.mode.changed`, `.deviation.reported`, `.inspection.requested`, `.action.observed` |
| `machine_state` | v1: `machine_state`; v2 (текущая): `execution` + `condition` + `controller_mode`, все обяз. | `equipment.state.changed` |
| `evidence_refs` | `evidence_refs[]` — адреса материалов (внутри обяз. `material_address`, `media_type`, `kind`, `is_illustration`) | только `inspection.result.recorded` и `operator.action.observed`; необяз. |
| `analyzer_version` | `versions.analyzer_version` — в векторе версий (AD-29); версия ступени — `stages[].version` | `inspection.result.recorded`, `operator.action.observed`: `versions` необяз., но если есть — обяз. `recipe_ref`, `analyzer_version`, `contract_version` |
| FR-27: `inspection_method`, `processing_state`, `recipe_ref`, `limitations[]`, `identification_level` | `method`, `processing_state`, `versions.recipe_ref`, `limitations[]`, `item_ref.identification_level` | конверт и `data` |
| FR-27: метаданные целостности | `integrity` | конверт, обяз. |

1. **`received_at`.** Время получения ставит приём по доменным часам: `backend/internal/application/ingest/pipeline.go`, функция `process` — `received` из `DomainClock.Now`, затем `ReceivedAt: received` в записи журнала. Если источник сам прислал `received_at` в конверте, приём его не читает (поля нет в структуре `header` того же файла) и не отвергает (схема конверта открыта, FR-29): оно молча остаётся в исходном конверте, который журнал хранит без изменений, и на время и порядок не влияет. Поле входит в отпечаток содержимого: повтор того же `event_id` с другим `received_at` — конфликт `ingest.duplicate_conflict`.
2. **`item_type_id`.** Тип изделия берётся из `item.item.registered` этого изделия; так делают, например, показатели: `backend/internal/domain/analytics/contribute.go` (`apply`: `item.item.registered` → `itemType`; `base()` добавляет тип в каждую строку).
3. **Линия и участок результата контроля.** В `inspection.result.recorded` есть `operation_run_id`, `step_key`, `inspection_point`, но нет `line_id`/`station_id`. Выполнение операции — указанный `operation_run_id`, а без него — последнее `operation.run.started` изделия до `occurred_at` результата («до следующей операции», FR-37). Шаг — из `step_key` результата, а без него — шаг выполнения; линия, участок, операция и исполнитель — из этого `operation.run.started`: их получают признаки дефектов и несоответствия, выведенные из результата, а строки самого результата контроля несут шаг, средство контроля и контролёра (`backend/internal/domain/analytics/contribute.go`: `runAt`, `inspection`, `finish` → `runDims`). `inspection_point` — точка контроля процесса (шаг с `ant:inspection` в BPMN). У камеры VisionQC точку, шаг и фазу задаёт конфигурация установки по `camera_id` (`backend/internal/infrastructure/integration/vision/visionqc/adapter.go`, `DefaultStations`). Перевод внешнего формата VisionQC в `inspection.result.recorded` поле за полем — [vision-camera-project.md, §10.2](../../docs/vision-camera-project.md#перевод-результата-visionqc-в-событие-контракта).

### Обязательные поля по типам

Конверт у всех типов: обяз. `event_id`, `event_type`, `schema_version`, `source_id`, `occurred_at`, `correlation_id`, `causation_id` (`null` у корневого), `integrity`, `data`. Схема **не требует** `source_seq`, `source_kind`, `reliability`, `item_id`, `item_ref`, `run_id`. Через приём входят только факты; реакции пишет ядро (`backend/internal/domain/ingest/contract.go`, `ClassifyVersion`).

| Тип | Вид | Поток | Обязательные поля `data` |
|---|---|---|---|
| `inspection.result.recorded` | факт | item | `method`, `phase`, `outcome`, `processing_state` |
| `inspection.point.mode_changed` | факт | equipment | `inspection_point`, `mode` |
| `operation.run.started` | факт | item | `operation_run_id`, `operation_code`, `step_key`, `operator_id` (может быть `null`) |
| `operation.run.paused` | факт | item | `operation_run_id`, `pause_reason` |
| `operation.run.resumed` | факт | item | `operation_run_id` |
| `operation.run.finished` | факт | item | `operation_run_id`, `completion` |
| `operation.movement.sent` | факт | item | `from_location_id`, `to_location_id`, `sent_by` |
| `operation.movement.received` | факт | item | `to_location_id`, `destination_kind`, `inspection_on_receipt`, `received_by` |
| `operation.run.interval_resolved` | реакция | item | `operation_run_id`, `interval_start`, `interval_origin` |
| `operation.precondition.failed` | реакция | item | `step_key`, `precondition`, `mode` |
| `operation.message.thrown` | реакция | item | `step_key`, `message_ref`, `erp_action`, `closing_basis` |
| `operator.step.confirmed` | факт | item | `operator_id`, `step_key` |
| `operator.check.skipped` | факт | item | `operator_id`, `step_key` |
| `operator.override.performed` | факт | item | `operator_id`, `bypassed` |
| `operator.mode.changed` | факт | item | `operator_id`, `equipment_id`, `change` |
| `operator.deviation.reported` | факт | item | `operator_id`, `description` |
| `operator.inspection.requested` | факт | item | `operator_id`, `step_key` |
| `operator.action.observed` | факт | item | `observation` |
| `equipment.state.changed` v2 | факт | equipment | `equipment_id`, `execution`, `controller_mode`, `condition` (v1: `equipment_id`, `machine_state`) |
| `equipment.program.changed` | факт | equipment | `equipment_id`, `program_ref`, `program_revision`, `planned` |
| `equipment.tool.changed` | факт | equipment | `equipment_id`, `tool_id` |
| `equipment.cycle.summarized` | факт | equipment | `equipment_id`, `window_start`, `window_end`, `parameters` |
| `equipment.deviation.detected` | факт | equipment | `equipment_id`, `deviation_kind`, `started_at` |
| `equipment.violation.window_resolved` | реакция | equipment | `equipment_id`, `window_start`, `window_end`, `deviation_event_ids`, `affected_operation_run_ids` |
| `equipment.event.bound` | реакция | item | `operation_run_id`, `equipment_id`, `binding`, `subject_event_id`, `subject_event_type`, `subject_occurred_at`, `subject_data` |

- **Изделие необязательно.** `item_id` и `item_ref` схема не требует. Событие с разрешённым изделием идёт в поток `item:‹item_id›`. Без изделия (нет ни того, ни другого, носитель не найден или неоднозначен) тип с потоком `item` идёт в поток `global` — партицию межизделийной стадии, которая привяжет его позже (AD-41); туда же — его флаги приёма. Тип с потоком `equipment` без изделия — в `equipment:‹equipment_id›`, а если в `data` нет `equipment_id` — в `source:‹source_id›` (`backend/internal/application/ingest/pipeline.go`: `streamOf`; флаги — в `process`).
- **`source_seq`, `source_kind`, `reliability`** схема не требует, и приём их отсутствие не отвергает; «для устройств» и «для фактов» в таблице конверта — соглашение для источников. Edge-агент ставит `source_id` и `source_seq` сам (`backend/cmd/edge-agent/agent.go`, `Enqueue`).
- **Без `source_seq` последовательность не проверяется:** разрывы номеров, досылка и повтор номера у такого сообщения не учитываются (`backend/internal/domain/ingest/sequence.go`, `ObserveSeq`: номер ≤ 0 → `SeqNone`).

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

- Факт обязан нести `source_kind` и `reliability`; реакция — блок `reaction`; решение — блок `command`. Это правило каталога: схема конверта и приём наличие `source_kind` и `reliability` не проверяют ([выше](#обязательные-поля-по-типам)).
- Запись с `corrects` — критическое действие группы `protected_data` независимо от типа (FR-122, AD-28).
- Разрешающее действие не может опираться только на факты `server-attested` (AD-2); справочники, от которых зависят права и предусловия, не меняются фактами `server-attested`.
- Классы происхождения спайна в контракте пишутся snake_case: `server-attested` → `server_attested`; остальные совпадают (`device`, `personal`, `paper`, `partner`, `scenario`, `genesis`).

## Сведение имён процессной сессии с каталогом v1

Одним шагом при фиксации контракта (эпик 00; источник — `process/process-events-catalog.md` процессной сессии, черновик). Смысл событий — процессной сессии, имена, поля и обработка — этого каталога; в код попадают только имена справа. Номера Е-‹n› и П-‹n› — из их каталога; у каждого типа они же указаны в `refs` в `catalog.yaml`.

### События

| Процессная сессия | Слой | Каталог v1 | Примечание |
|---|---|---|---|
| Е-01 `erp.order_received` | И | `erp.order.received` | старт процесса по сообщению (`triggerEventType`, Д-5) |
| Е-02 `erp.nomenclature_synced` | И | `erp.nomenclature.synced` | |
| Е-02 `assembly.structure_imported` | И | `cad.assembly.imported` | файл условной сборки КОМПАС |
| Е-03 `erp.batch_received` | И | `erp.lot.received` | «партия» в коде — `lot` |
| Е-04 `batch.registered` | Ф | `genealogy.lot.registered` | |
| Е-05 `inspection.result` (метод «документы») | Ф | `inspection.result.recorded` (`method = supplier_documents`) | |
| Е-06, Е-17, Е-18, Е-44, Е-45, Е-52, Е-54, Е-62, Е-70 `inspection.result` | Ф | `inspection.result.recorded` | один тип для всех методов (FR-36); `defect_found` → `defect_indicated`, `no_defect_found` → `no_defect_indicated`, `unable_to_assess` — без изменений |
| Е-07 `item.marked` | Ф | `item.carrier.applied` | перемаркировка — `replaces_value` |
| Е-08 `decision.gate_passed` (ЗТ-1, партия) | Р | `decision.lot.resolved` | |
| Е-19, Е-47, Е-55, Е-63, Е-73 `decision.gate_passed` | Р | `decision.presentation.resolved` | `resolution` — переменная `decision` в условиях BPMN |
| Е-09 `item.registered` | Ф | `item.item.registered` | решение (запуск), закрепляет версию процесса |
| Е-10, Е-40, Е-60 `operation.started` | Ф | `operation.run.started` | |
| Е-11 `operation.paused` / `operation.resumed` | Ф | `operation.run.paused` / `operation.run.resumed` | |
| Е-12 `operation.finished` | Ф | `operation.run.finished` | длительность — со смыслом и происхождением |
| Е-13 `operation.precondition_failed` | В | `operation.precondition.failed` | |
| Е-14 `machine.state` (`running`/`idle`/`stopped`) | Ф | `equipment.state.changed` (v2: `execution`) | v1 с `machine_state` — для чтения старых записей |
| Е-15, Е-43 `machine.state` (`warning`) | Ф | `equipment.deviation.detected` + `equipment.state.changed` (`condition = warning`) | отклонение — самостоятельный сигнал (FR-121) |
| Е-16 `machine.state` (`alarm`) | Ф | `equipment.state.changed` (`condition = fault`) + `equipment.deviation.detected` (`alarm`) | |
| Е-42, Е-53, Е-61 `machine.parameters` | Ф | `equipment.cycle.summarized` | сводка на окно цикла, не телеметрия |
| Е-20 `operator.action` (`mode_change`) | Ф | `operator.mode.changed` | критическое действие |
| Е-21 `operator.action` (`check_skipped`) | Ф | `operator.check.skipped` | критическое действие |
| Е-22 `operator.action` (`manual_override`) | Ф | `operator.override.performed` | критическое действие |
| Е-23 `operator.action` (`confirmation`) | Ф | `operator.step.confirmed` | |
| Е-24 `operator_action.detected` | Ф | `operator.action.observed` | гипотеза OperatorVision |
| Е-30 `movement.sent` | Ф | `operation.movement.sent` | |
| Е-31 `movement.received` | Р | `operation.movement.received` | у нас — факт с личной подписью мастера (уровень 1) |
| Е-41, Е-51 `assembly.component_linked` | Ф | `item.assembly.recorded` → `genealogy.link.added` | связь генеалогии — реакция стадии |
| Е-46 `rework.limit_reached` | В | `operation.precondition.failed` (`precondition = rework_limit`); разрешение сверх лимита — `decision.rework_limit.waived` | гард — `process.rework_limit_exceeded` |
| Е-50 `batch.issued` | Ф | `genealogy.lot.issued` | |
| Е-71 `item.inspection_summary` | В | — (проекция `quality`, не запись журнала) | гард ЗТ-6 — `nonconformity.method_result_missing` |
| Е-72 `item.presented` | Ф | `item.presentation.recorded` | |
| Е-74 `item.released` | Ф | `item.release.recorded` | |
| `erp.posting_sent` | И | `operation.message.thrown` → `erp.posting.requested` | бизнес-ключ идемпотентности (AD-7) |
| Е-75 `integration.ack` / `integration.error` | И | `erp.posting.responded` (`outcome`) | ось «учёт в 1С» — только по подтверждению |
| Е-76 `erp.control_result_sent` [П процессной сессии] | И | `erp.posting.requested` (`action = inspection_result`) → `erp.posting.responded` | «результат контроля» уже есть в порте учёта (AD-18, кейс §3.3); изделие не держит |
| Е-80 `quality.signal_raised` | В | `quality.signal.raised` | |
| Е-81 `quality.observation_linked` | В | `quality.observation.linked` (новый дефект — `quality.defect.identified`) | |
| Е-82 `quality.inspection_missing` | В | `quality.inspection.missing` | |
| Е-83 `nc.draft_created` | В | `decision.nonconformity.drafted` | |
| Е-84 `containment.hold_applied` | В | `decision.containment.applied` | |
| Е-85 `analysis.hypotheses_updated` | В | `incident.hypothesis.computed` | |
| Е-86 `checkpoint.mode_changed` | Ф | `inspection.point.mode_changed` | `checkpoint` в коде — только контрольная точка журнала |
| Е-90 `decision.signal_confirmed` | Р | `decision.nonconformity.confirmed` | |
| Е-91 `decision.signal_rejected` | Р | `decision.signal.rejected` | |
| Е-92 `decision.recheck_requested` | Р | `decision.recheck.requested` | |
| Е-93 `decision.item_isolated` | Р | `decision.item.isolated` | |
| Е-93 `movement.isolator_confirmed` | Ф | `operation.movement.received` (`destination_kind = isolator`) | |
| Е-94 `decision.disposition_set` | Р | `decision.disposition.set` | типовая переделка по правилу режима 2 — `decision.disposition.applied` |
| Е-95 `permit.approved` | Р | `document.signature.recorded` → `document.route.closed` (шаблон `concession`) | разрешение действует после закрытия маршрута |
| Е-95 `permit.revoked` | Р | `decision.concession.revoked` | |
| Е-96 `decision.disposition_verified` | Р | `decision.disposition.verified` | |
| Е-97 `nc.disposition_overdue` | В | `obligation.due.reached` → `obligation.escalation.raised` | сроки эмитит только notifications (AD-4) |
| Е-98 `nc.item_closed` | Р | `decision.nonconformity.closed` | |
| Е-100 `event.corrected` | Р | запись того же типа с блоком `corrects` в конверте | критическое действие `protected_data` |
| Е-101 `item.binding_corrected` | Р | `binding.link.assigned` | |
| Е-110 `risk_scope.created` | В | `incident.incident.opened` + `incident.scope.computed` + `incident.membership.changed` | |
| Е-111 `risk_scope.expanded` | В / Р | правилом — `incident.scope.computed`; человеком — `incident.scope.expanded` | один тип — один вид записи |
| Е-112 `risk_scope.narrowed` | Р | `incident.scope.narrowed` | |
| Е-113 `risk_scope.item_assessed` | Р | `incident.item.assessed` | |
| Е-114 `process_hold.set` / `process_hold.released` | Р | `decision.process_hold.set` / `decision.process_hold.released` | |
| Е-115 `risk_scope.closed` | Р | `incident.incident.closed` | |
| Е-120 `intervention.opened` | Р | `item.intervention.opened` | |
| Е-121 `intervention.closed` | Р | `item.intervention.closed` | |
| П-01 `quality.escape_recorded` | В | `quality.escape.recorded` | |
| П-02 `cause.analysis_scoped` | Р | `incident.analysis.scoped` | |
| П-03 `cause.hypothesis_recorded` | В / Р | системой — `incident.hypothesis.computed`; человеком — `incident.hypothesis.recorded` | |
| П-04 `cause.confirmed` | Р | `incident.cause.concluded` (`conclusion = confirmed / not_established`); ошибка исполнителя — `incident.operator_error.confirmed` | |
| П-05 `capa.action_assigned` | Р | `incident.action.assigned` | |
| П-06 `capa.action_implemented` | Ф | `incident.action.implemented` | |
| П-07 `capa.effectiveness_confirmed` / `capa.effectiveness_failed` | Р | `incident.action.evaluated` (`result`) | |
| П-08 `norm.version_approved` | Р | `normative.version.activated` | после закрытия маршрута кворума |

Сведено (Д-17, Д-58): «возврат из брака в производство» после удачной переделки на повторной ЗТ — учётное действие `return_from_defect` порта учёта AD-18: значение `action` в `operation.message.thrown` / `erp.posting.requested` и `erpAction` в `contracts/bpmn-ext/rules.yaml`; код — `backend/internal/domain/erp/action.go`, описание — [docs/integrations/1c.md](../../docs/integrations/1c.md).

### Коды ошибок процессной сессии

| Рабочий код | Код контракта |
|---|---|
| `E_MISSING_FIELD` | `ingest.missing_required_field` |
| `E_UNSUPPORTED_VERSION` | `ingest.unknown_schema_version` |
| `E_UNKNOWN_ENUM` | `ingest.unknown_enum_value` (важное для безопасности поле — `ingest.unknown_enum_value_critical`) |
| `E_ID_CONFLICT` | `ingest.duplicate_conflict` |
| `E_REWORK_LIMIT` | `process.rework_limit_exceeded` (решение Д-9) |
| `E_PERMIT_REQUIRED` | `nonconformity.concession_required` (решение Д-9) |
| `E_REF_NOT_FOUND` | `reference.not_found` (решение Д-9) |

Сведение хранится и в `contracts/errors.yaml` (поле `aliases`).

### Статусы

Коды статусов — `contracts/statuses.yaml`. Ключи текстов интерфейса процессной сессии (`statuses.*` в `ui-texts/ru.json`) переводятся в коды заменой camelCase на snake_case (`statuses.position.inQueue` → `position.in_queue`) со следующими отличиями:

- `statuses.containment.processPointStop`, `criticalStop` → словарь `process_containment` (сдерживание процесса — отдельный объект, FR-49); в оси «сдерживание» блок изделия и блок партии — `item_hold`, `lot_hold`.
- `statuses.erpAccounting.sentAwaitingAck`, `exchangeError` → словарь `erp_message_status` (`sent`, `rejected`): ось «учёт в 1С» — только шесть значений §3b и меняется только квитанцией.
- `statuses.itemSummary.*` → словарь `item_summary` (сводный статус, вычисляется из осей).
- `statuses.quality.signal` → `quality.signal` («Сигнал о признаке дефекта»).
- Переменные условий BPMN процессной сессии (`decision`, `nc.outcome`, `disposition`, `test.result`, `tools.accounted`, `plan.radiography_required`) сохранены как есть — `contracts/bpmn-ext/rules.yaml`.

# Контракт обмена с MES — mes.isa95.v1 (эпик 31)

Подмножество транзакций **B2MML V7** (ISA-95 / IEC 62264, MESA International) в форме B2MML-JSON (FR-93, AD-18;
проектное предположение): имена элементов и атрибутов — как в XSD B2MML, атрибуты XML — свойствами
(`releaseID`, `versionID`, `actionCode`). Источник правды для адаптера
`backend/internal/infrastructure/integration/mes/b2mml` (порт `application/mes.Channel`); описание обмена —
`docs/integrations/mes.md`. Схемы `b2mml/` — внешний стандарт: линтер схем не требует у них snake_case
(как у `payloadType` DSSE, `contracts/scripts/lint-schemas.mjs`).

| Файл | Направление | Что это |
|---|---|---|
| `b2mml/ProcessOperationsSchedule.schema.json` | MES → Главный | задания: расписание → запросы → требования сегментов (`ProcessSegmentID` — код операции нормативного слоя) → `mes.job.received` |
| `b2mml/NotifyOperationsEvent.schema.json` | MES → Главный | события операций `Start`, `End`, `Pause`, `Resume`, `Abort` над экземпляром → `operation.run.started` / `finished` / `paused` / `resumed` того же изделия, что и у терминала (FR-140) |
| `b2mml/SyncMaterialSubLot.schema.json` | Главный → MES | блок (`Disposition = Restricted`, `Status = QC-HOLD`) и снятие (`UnRestricted`) экземпляра; `ConfirmationCode = Always` |
| `b2mml/SyncMaterialLot.schema.json` | Главный → MES | то же для партии |
| `b2mml/ConfirmBOD.schema.json` | в обе стороны | подтверждение: успех (повтор того же `BODID` — `Duplicate`), `ErrorMessage` с `ErrorType` |
| `b2mml/common.schema.json` | — | `ApplicationArea` (`BODID` — ключ идемпотентности), количество, идентификаторы |
| `binding/about.schema.json`, `binding/outbox.schema.json` | — | HTTP-привязка: `GET ‹канал›/about` (релиз B2MML и версии контракта mes.isa95: `mes.isa95.v1`, …), `GET ‹канал›/outbox` (сообщения MES для Главного) |
| `examples/` | — | эталоны; исходящие и `ConfirmBOD` формирует адаптер (контрактный тест сравнивает с эталоном) |

**HTTP-привязка (MVP):** `POST ‹канал›/SyncMaterialSubLot`, `POST ‹канал›/SyncMaterialLot` — ответ `ConfirmBOD`;
`GET ‹канал›/outbox`, `GET ‹канал›/about`. Брокер — сменный адаптер того же порта.

**Классы ответа.** `ConfirmBOD` с успехом — `mes.hold.responded accepted` (повтор — `duplicate`, второго блока
нет); `ErrorType = Data` — `rejected` с кодом MES, без автоповтора, изделие остаётся заблокированным в Главном;
`ErrorType = Contract | Security`, `401`/`403`, `about` без `mes.isa95.v1` — несовместимость, канал `degraded`;
`5xx`, `408`, `429`, таймаут, `ErrorType = Transport` — повтор с тем же `BODID`. Исходящее сообщение до
отправки проверяется схемой (FR-111); входящее разбирается терпимо (лишние элементы стандарта отбрасываются)
и проверяется схемой подмножества.

**Сопоставление ID (FR-95).** Экземпляр, сотрудник, оборудование MES → наши ID только по записям
`reference.external_id.mapped` (система `mes`); MES может хранить и наш ID изделия, полученный в блоке
(`MaterialSubLot.ID` = `ENT01:…`). Нет соответствия или код операции не найден в BPMN — сообщение
откладывается с причиной, факт не выдумывается (FR-123). `event_id` — от `BODID` и ID события: повторная
доставка — дубль (AD-7).

**Контрактный тест:** `go test ./internal/infrastructure/integration/mes/...` — эталоны по схемам, блок →
эталонные сообщения, классы `ConfirmBOD`, шлюз входящих с BPMN фланца и схемами приёма, сквозной путь
«сдерживание → `mes.hold.requested` → MES → `mes.hold.responded` → `mes.block.list`». Stand MES — эпик 43.

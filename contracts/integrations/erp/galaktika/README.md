# Контракт обмена с Галактика ERP — gal.qc.v1 (эпик 31)

Второй получатель порта учёта (FR-92, AD-18, AD-35): тот же внутренний сигнал, что уходит в 1С
(`application/erp.Ledger`), адаптер `backend/internal/infrastructure/integration/erp/galaktika` переводит в пакет
Галактики. Ядро, домен `erp`, роль `outbox` и контракт событий не меняются — включённая система задаётся
конфигурацией (`docs/new-adapter.md`). Описание обмена — `docs/integrations/galaktika.md`. **Все форматы —
проектные предположения** (публичной спецификации API Галактики нет, NFR-DOC-2).

Платформа для людей — **Главный** (техническое имя `ant`, Д-65).

**Одна модель — два транспорта** (ключ `transport`):

| Транспорт | Что передаётся | Где |
|---|---|---|
| `exchange-dir` (основной для 9.x) | XML-пакет `GalExchange` (`gal.qc.v1.xsd`) | `out/‹номер›.xml` — от Главного; `ack/‹номер›.xml` — квитанция Галактики; `in/*.xml` — пакеты Галактики; `in-ack/‹номер›.xml` — квитанция Главного; `about.xml` — версии контракта обработчика |
| `rest-facade` («под Галактика ESB») | JSON-форма того же пакета (`gal.qc.v1/exchange.schema.json`) | `POST ‹фасад›/quality/lot-results`, `POST ‹фасад›/production/postings` (заголовки `X-Message-Id`, `X-Contract-Version: gal.qc.v1`), `GET ‹фасад›/exchange/outbox`, `GET ‹фасад›/about` |

| Файл | Что это |
|---|---|
| `gal.qc.v1.xsd` | XML-схема пакета, квитанции `GalAck` и описания `GalAbout` — для обработчика на стороне Галактики (VIP / объектный интерфейс «Атлантиса») |
| `gal.qc.v1/exchange.schema.json` | JSON-форма пакета; ею адаптер проверяет **каждое** исходящее сообщение до отправки на обоих транспортах (FR-111) |
| `gal.qc.v1/ack.schema.json`, `about.schema.json`, `inbox.schema.json` | квитанция, описание ответной стороны, входящие пакеты фасада |
| `examples/*.xml`, `examples/*.json` | эталоны: XML- и JSON-формы одного пакета совпадают (контрактный тест) |

**Имена XML ↔ JSON.** Элементы — `PascalCase`, атрибуты — `camelCase`, JSON — `snake_case`:
`GalExchange/@messageId` ↔ `message_id`, `QualityLotResult` ↔ `quality_lot_result`, `Posting` ↔ `posting`,
`ProductionTask` ↔ `production_task`, `LotReceived` ↔ `lot_received`, `ItemCatalog` ↔ `item_catalog`,
`Source/@businessKey` ↔ `source.business_key`, `Nonconformity/@ref` ↔ `nonconformities[].ref`. Значения
перечислений одинаковы в обеих формах.

**Порт учёта → gal.qc.v1.** `inspection_result` → `QualityLotResult` (`verdict`: `accepted`,
`accepted_with_concession`, `accepted_partially`, `rejected`, `insufficient_data`); прочие действия →
`Posting/@kind`: `accept_into_work` → `accept_to_work`, `warehouse_transfer` → `internal_move`,
`scrap_transfer_rework|writeoff|reprocess` → `defect_rework|defect_writeoff|defect_reprocess`,
`return_to_supplier` → `return_to_supplier`, `return_from_defect` → `return_from_defect`, `release` → `release`
(`afterRework`, `concession`).

**Классы ответа.** `GalAck status="ok"` — квитанция (`202`; повтор того же номера — `200` и `duplicate="true"`,
второй документ не создаётся, AD-7); `status="error"` — ошибка данных (`422`, без автоповтора → карантин
исходящих; `LOT_NOT_FOUND`, `ITEM_NOT_MAPPED`, `WAREHOUSE_NOT_FOUND`, `ORDER_NOT_FOUND` → `erp.id_mapping_missing`,
`CONTRACT_NOT_FOUND` → `erp.contract_not_found`, прочее → `erp.data_error`); `CONTRACT_VERSION`,
`SCHEMA_VIOLATION`, `401`/`403` — несовместимость, канал `degraded`; `5xx`, `408`, `429`, `409`, таймаут и «квитанции
в `ack/` ещё нет» — транспорт, повтор с тем же номером пакета. `about` без `gal.qc.v1` среди `supported` —
канал `degraded`, отправки нет (AD-18).

**Входящие → факты порта учёта** (через обычный приём, источник `erp.galaktika`):
`ItemCatalog` → `erp.nomenclature.synced`; `ProductionTask` → `erp.order.received` (наш ID `ORD-‹номер латиницей›`);
`LotReceived` → `erp.lot.received` (`LOT-‹номер партии›`); соответствия — `reference.external_id.mapped`
системы `galaktika` с внешним ID `galaktika:‹база›:‹таблица›:‹NRec›` (NRec — строкой). Тип изделия — по
обозначению КД (`domain/cad.ItemTypeID`), как у 1С и КОМПАС: три системы приходят к одному нашему типу.

**Контрактные тесты:** `go test ./internal/infrastructure/integration/erp/...` — эталоны по схемам и XSD,
сигнал порта → эталонные пакеты, общий контрактный тест порта учёта `application/erp/ledgertest` на обоих
транспортах Галактики, на stand-е 1С и на stand-е Галактики (эпик 43, `…/erp/galaktika/stand`).

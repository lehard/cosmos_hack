# Контракт обмена с 1С (эпик 30)

Внешний контракт порта учёта для 1С:ERP (AD-18, AD-20, FR-90, FR-91, FR-96, FR-111). Источник правды для
адаптера `backend/internal/infrastructure/integration/erp/onec` и stand-а 1С
`…/erp/onec/stand`; описание обмена — `docs/integrations/1c.md`.

| Файл | Что это |
|---|---|
| `qc.v1/posting.schema.json` | тело `POST /‹база›/hs/qc/v1/postings` — учётное действие на закрывающей точке |
| `qc.v1/inspection-result.schema.json` | тело `POST /‹база›/hs/qc/v1/inspection-results` — результат контроля |
| `qc.v1/receipt.schema.json` | квитанция: `202` — принято, `200` — повтор с тем же `X-Message-Id`, та же квитанция |
| `qc.v1/error.schema.json` | ошибка: `422` данные (карантин), `400` контракт, `503` недоступность (повтор) |
| `qc.v1/about.schema.json` | `GET /‹база›/hs/qc/v1/about` — версия контракта расширения (сверка при старте) |
| `qc.v1/common.schema.json` | общие определения: источник и бизнес-ключ, изделие, партия, склад, документ |
| `metadata-manifest.json` | ожидаемые сущности и реквизиты OData v3 (`$metadata`) — сверка при старте; stand строит из него свой `$metadata` |
| `examples/` | эталонные сообщения; контрактный тест адаптера сверяет их со схемами (`go test ./internal/infrastructure/integration/erp/onec/...`) |

**Заголовки записи:** `X-Message-Id: ‹UUID›` — ключ идемпотентности (UUIDv5 от бизнес-ключа и версии
содержимого, AD-7), `X-Contract-Version: qc.v1`, `Content-Type: application/json`.

**Классы ответа и поведение `ant`:** `202` — квитанция, ось «учёт в 1С» меняется; `200` — повтор, та же
квитанция, двойного учёта нет; `422` — ошибка данных, без автоповтора → карантин исходящих, ручная
переотправка; `400 CONTRACT_VERSION` / расхождение `$metadata` — канал `degraded`, отправка остановлена;
`503`, `5xx`, таймаут, разрыв — повтор с тем же номером сообщения с растущей задержкой, затем карантин.

**Словарь учётных действий** (наш язык → `kind` qc.v1): `accept_into_work` → `accepted_to_work`;
`warehouse_transfer` → `warehouse_transfer`; `scrap_transfer_rework|writeoff|reprocess` → `moved_to_defect`
(`defect_kind` = `rework|scrap|reprocessing`); `return_to_supplier` → `returned_to_supplier`;
`return_from_defect` → `returned_from_defect` (решение Д-17); `release` → `released`;
`inspection_result` → отдельный ресурс `inspection-results`.

Имена реквизитов OData и наш сервис `qc` — проектное предположение до сверки с базой предприятия (NFR-DOC-2).

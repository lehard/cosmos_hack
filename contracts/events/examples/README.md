# Примеры сообщений контракта

## Пять случаев изменения контракта (FR-29, кейс §4.7, AD-20)

Каждый файл `contract-change/*.json` — `{case, note, expect, message}`; `contracts/scripts/check-examples.mjs` прогоняет `message` через эмуляцию правил приёма (те же схемы, что у настоящего приёма, эпик 06) и сверяет с `expect`.

| Файл | Случай | Поведение | Код |
|---|---|---|---|
| `01-unknown-version.json` | неизвестная версия (`schema_version` 3) | карантин с кодом; переобработка после появления схемы и повышателя | `ingest.unknown_schema_version` |
| `02-new-optional-field.json` | новое необязательное поле в `data` и в конверте | принимается, исходник сохраняется целиком (схемы событий открыты) | — |
| `03-missing-required-field.json` | нет обязательного поля `outcome` | problem+json с кодом и карантин | `ingest.missing_required_field` |
| `04a-unknown-enum-value.json` | неизвестное значение перечисления (`pause_reason`) | принимается как `UNKNOWN(значение)` с флагом (`ingest.anomaly.flagged`) | `ingest.unknown_enum_value` |
| `04b-unknown-enum-value-critical.json` | неизвестное значение в поле, важном для безопасности (`outcome`) | карантин — «годно» не додумывается; перечень полей — `../safety-critical-enums.yaml` | `ingest.unknown_enum_value_critical` |
| `05-incompatible-change.json` | несовместимое изменение без новой мажорной версии | карантин; правильно — новая мажорная версия (см. ниже); в сборке ломающее изменение ловит проверка контракта (`make check`, эпик 02) | `ingest.missing_required_field` |

## Две последовательные версии одного типа (FR-110)

`equipment.state.changed`:

- **v1** (`../equipment/equipment.state.changed.v1.json`) — одно поле `machine_state` (`running | idle | stopped | warning | alarm`), как в кейсе §4.4 и в каталоге процессной сессии; смешивает режим работы и исправность.
- **v2** (`../equipment/equipment.state.changed.v2.json`) — по классификации MTConnect (AD-29, FR-147): `execution`, `controller_mode`, `condition`. Изменение несовместимое (удалено обязательное поле, добавлены обязательные) — поэтому новая мажорная версия.
- Повышатель v1 → v2 — `../upcasters/equipment.state.changed.v1-to-v2.yaml` (декларативная таблица, из неё эпик 02 генерирует чистую функцию); чего в v1 нет — `unknown`.
- Примеры: `versions/equipment.state.changed.v1.json`, `.v2.json` и `…v1-upcast-to-v2.json` — результат повышения, проходит схему v2.

Воспроизводимое изменение для материалов кейса §6.2 (генерация и обнаружение до отправки в 1С) — `make contract-demo`, эпик 02.

# Интеграция с MES

Обмен с системой управления производством (MES) без повторного ручного ввода (FR-93, кейс §3.3): из MES в `ant` приходят задания и события операций, из `ant` в MES — блокировки изделий и партий (чтобы заблокированное изделие не получило следующую операцию), результаты контроля и задания на доработку. Контракт — нейтральный, по ISA-95 / IEC 62264 в форме B2MML. В MVP — адаптер в коде, контракт `mes.isa95.v1`, эталонные сообщения и контрактный тест; stand MES — в очереди работ.

**Опоры:** FR-90, FR-93, FR-95, FR-96; AD-7, AD-18, AD-20, AD-30; кейс §3.3, §4.3, §5.3; критерии Т2, О8. Общие правила — [README.md](README.md).

## 1. Какая MES и почему нейтральный контракт

На месте «MES» у предприятия может стоять разное (по открытым материалам вендоров):

| Система | Что известно | Интерфейс |
|---|---|---|
| 1С:ERP — пооперационное планирование («MES») и 1С:MES ОУП | задания, этапы, выполнение операций, учёт брака; обмен с 1С:ERP через EnterpriseData с квитированием | через адаптер 1С / EnterpriseData |
| Галактика MES | сменные задания, учёт исполнителем, электронный паспорт, прослеживаемость | через Галактика ESB |
| ГОЛЬФСТРИМ (АСКОН) | MES на платформе ЛОЦМАН:PLM; сопроводительные листы, наряды, акты о браке, работа ОТК | ЛОЦМАН API / интеграционная шина (XML) |
| Самописные цеховые системы | цеховой учёт в собственных разработках | нужен нейтральный контракт |

Единого российского стандарта обмена с MES нет, поэтому `ant` говорит на международном нейтральном: **ISA-95 / IEC 62264**, сообщения — **B2MML** от MESA International (XSD, релиз V7, есть JSON-вариант, сгенерированный из XSD) (проверено по документации). Адаптер конкретной MES переводит B2MML в её формат; наш контракт `mes.isa95.v1` — **проектное предположение**: подмножество транзакций B2MML в форме B2MML-JSON (AD-18).

## 2. Модель сообщений B2MML (проверено по документации)

- Имя сообщения — `‹Глагол›‹Существительное›`: `ProcessOperationsSchedule`, `SyncMaterialLot`, `NotifyOperationsEvent`, `ProcessTestResult`.
- Обёртка: `ApplicationArea` (`Sender`, `Receiver`, `CreationDateTime`, **`BODID`** — идентификатор сообщения) + `DataArea` (глагол + существительные); на корне — `releaseID` / `versionID`.
- Подтверждения: `ConfirmBOD` (`ConfirmationCode` = `Always | Never | OnError`), ответы `Acknowledge*`; ошибки — `ErrorMessage` (`ErrorCode`, `ErrorDescription`, `ErrorType`).
- Состояние партии — `Disposition`: `Planned`, `In-Process`, **`Restricted`** («не разрешена к нормальному использованию; пример — партия ожидает решения по качеству»), `UnRestricted`, `Closed` («израсходована, продана или утилизирована»), `Other`. Наши уровни сдерживания ложатся на стандарт без натяжки.

## 3. Направления, данные и шаги процесса

| Событие | Сообщение B2MML | Направление | Подтверждение | Шаг процесса |
|---|---|---|---|---|
| задание / партия запуска выдана | `ProcessOperationsSchedule` (`OperationsRequest` → `SegmentRequirement`: деталь, количество, маршрут) | MES → `ant` | `AcknowledgeOperationsSchedule` | запуск процесса изделий |
| операция начата / завершена / пауза | `NotifyOperationsEvent` (`Category`: начало, конец, остановка) или `SyncOperationsPerformance` | MES → `ant` | `ConfirmBOD` при `OnError` | каждая операция маршрута |
| блок изделия или партии (защитная реакция) | `SyncMaterialSubLot` / `SyncMaterialLot` с `Disposition=Restricted` + `WorkAlert` мастеру | `ant` → MES | `ConfirmBOD` с `Always` — блок должен быть подтверждён | на любом шаге: сигнал по карте реакций, блок партии, область риска |
| снятие блока (решение человека) | `SyncMaterialLot` `Disposition=UnRestricted` | `ant` → MES | `ConfirmBOD` | после решения на ЗТ-Р или сужения области риска |
| результат контроля | `ProcessTestResult` (`TestableObjectID` — экземпляр или партия, `EvaluatedCriterionResult`, `PropertyMeasurement`) | `ant` → MES | `AcknowledgeTestResult` | каждая подписанная ЗТ-1…ЗТ-6 |
| доработка / ремонт | `ProcessOperationsSchedule` с сегментом доработки и ссылкой на решение | `ant` → MES | `AcknowledgeOperationsSchedule` | ЗТ-Р: переделка, ремонт |
| окончательный брак | `SyncMaterialSubLot` `Disposition=Closed` | `ant` → MES | `ConfirmBOD` | ЗТ-Р: списать |
| поиск затронутых изделий | `GetMaterialLot` → `ShowMaterialLot` с `AssemblyLot` (генеалогия) | `ant` → MES → `ant` | ответ `Show` | область риска по партии (описание, не MVP) |
| ошибка обработки | `ConfirmBOD` с описанием / `ErrorMessage` | в обе стороны | — | — |

**Как блок согласован с моделью статусов (AD-30).** Блок — защитная реакция движка (ось «сдерживание», модуль `nonconformity`), MES только получает о нём сообщение. Если MES всё же прислала факт «операция начата» над заблокированным изделием, факт **принимается** с сигналом нарушения (внешний факт не отклоняется), затем реакция и повторный блок в MES. Команду API или терминала `ant`, нарушающую блок, гард отклоняет.

**«Без повторного ручного ввода».** Когда MES есть, факты операций приходят из неё, и исполнителю не нужно дублировать их на терминале `ant`; когда MES нет, те же факты даёт терминал исполнителя (FR-137). История изделия одна и та же, различается только `source_kind` факта (FR-140).

## 4. Источник достоверных сведений

| Данные | Источник истины |
|---|---|
| пооперационный план, сменные задания | MES (или ERP, если MES нет) |
| факты операций: начало, конец, исполнитель, оборудование | MES; без MES — терминал исполнителя `ant` |
| результаты контроля, сдерживание, решения по изделию | `ant` |
| генеалогия партий в MES (`AssemblyLot`) | MES — для своих партий; генеалогия изделий — `ant` (межизделийная стадия) |

## 5. Сопоставление идентификаторов

- Внешний ID — `ID` существительного: `OperationsRequest.ID`, `MaterialLot.ID`, `MaterialSubLot.ID`; ID сообщения — `BODID`.
- Партия MES ≠ серия ERP: связь «экземпляр `ant` ↔ партия MES ↔ серия 1С» хранится явно — несколькими записями `reference.external_id.mapped` на один наш ID.
- В исходящих `TestableObjectID` / `MaterialSubLot.ID` адаптер ставит ID MES, если соответствие есть, иначе — естественный ключ `‹обозначение›#‹заводской номер›`, а наш ID — в `Description` / расширении.
- Входящий `BODID` становится основой `event_id` факта (детерминированно), поэтому повторная доставка того же сообщения — дубль, а не новое событие (FR-31).

## 6. Контракт `mes.isa95.v1`

- Форма — B2MML-JSON (имена элементов по XSD V7), `releaseID="7.01"`, `versionID="mes.isa95.v1"`.
- Подмножество: `ProcessOperationsSchedule`, `AcknowledgeOperationsSchedule`, `NotifyOperationsEvent`, `SyncOperationsPerformance`, `SyncMaterialLot`, `SyncMaterialSubLot`, `ProcessTestResult`, `AcknowledgeTestResult`, `ConfirmBOD`.
- Транспорт в MVP — HTTP (`POST` сообщения на адрес канала MES, входящие — на операцию приёма `ant`); брокер — сменный адаптер за портом `Publisher` (см. [observability-kafka-otel.md](../observability-kafka-otel.md)).
- Схемы — `contracts/integrations/mes/b2mml/` (родные XSD MESA + JSON-схема подмножества), эталонные сообщения — `contracts/integrations/mes/examples/` (пути предварительные).
- Исходящие проверяются по XSD в контрактных тестах; входящие разбираются терпимо (лишние элементы не ломают разбор) и проверяются JSON-схемой подмножества на приёме.

## 7. Примеры сообщений

**1. Входящее задание:**

```json
{
  "ProcessOperationsSchedule": {
    "releaseID": "7.01", "versionID": "mes.isa95.v1",
    "ApplicationArea": {
      "Sender": {"LogicalID": "mes-ceh-2", "ConfirmationCode": "OnError"},
      "CreationDateTime": "2026-09-25T05:10:00.000Z",
      "BODID": "mes-2026-09-25-004411"
    },
    "DataArea": {
      "Process": {},
      "OperationsSchedule": [{
        "ID": "OS-2026-0925-01",
        "OperationsRequest": [{
          "ID": "OR-000812",
          "SegmentRequirement": [{
            "ID": "SR-1", "ProcessSegmentID": "МО-010 Мехобработка фланца",
            "MaterialRequirement": [{"MaterialDefinitionID": "ФЛ-100.00.001", "Quantity": {"QuantityString": "6", "UnitOfMeasure": "шт"}}]
          }]
        }]
      }]
    }
  }
}
```

**2. Исходящий блок экземпляра (защитная реакция):**

```json
{
  "SyncMaterialSubLot": {
    "releaseID": "7.01", "versionID": "mes.isa95.v1",
    "ApplicationArea": {
      "Sender": {"LogicalID": "ant", "ConfirmationCode": "Always"},
      "CreationDateTime": "2026-09-25T10:42:17.305Z",
      "BODID": "7c2d9e40-1a3b-5c4d-8e6f-7a8b9c0d1e2f"
    },
    "DataArea": {
      "Sync": {"ActionCriteria": {"ActionExpression": {"actionCode": "Change"}}},
      "MaterialSubLot": [{
        "ID": "ФЛ-100.00.000#0007",
        "MaterialLotID": "П-2026-0915",
        "Disposition": "Restricted",
        "Status": "QC-HOLD",
        "Description": "Блок по сигналу контроля; ant: ENT01:FL-0007; ожидается решение контролёра",
        "StorageLocation": "Изолятор ОТК, сварочный цех"
      }]
    }
  }
}
```

**3. Подтверждение:**

```json
{
  "ConfirmBOD": {
    "releaseID": "7.01",
    "ApplicationArea": {"Sender": {"LogicalID": "mes-ceh-2"}, "CreationDateTime": "2026-09-25T10:42:17.911Z", "BODID": "mes-ack-0098812"},
    "DataArea": {
      "Confirm": {},
      "BOD": [{"OriginalApplicationArea": {"BODID": "7c2d9e40-1a3b-5c4d-8e6f-7a8b9c0d1e2f"}, "BODSuccessMessage": {}}]
    }
  }
}
```

**4. Ошибка:**

```json
{
  "ConfirmBOD": {
    "releaseID": "7.01",
    "ApplicationArea": {"Sender": {"LogicalID": "mes-ceh-2"}, "CreationDateTime": "2026-09-25T10:42:18.020Z", "BODID": "mes-ack-0098813"},
    "DataArea": {
      "Confirm": {},
      "BOD": [{
        "OriginalApplicationArea": {"BODID": "7c2d9e40-1a3b-5c4d-8e6f-7a8b9c0d1e2f"},
        "BODFailureMessage": {"ErrorMessage": [{"ErrorCode": "SUBLOT_UNKNOWN", "ErrorType": "Data", "ErrorDescription": "Экземпляр ФЛ-100.00.000#0007 не найден"}]}
      }]
    }
  }
}
```

## 8. Подтверждения и обработка ошибок

| Ответ | Поведение `ant` |
|---|---|
| `ConfirmBOD` с успехом / `Acknowledge*` | квитанция → событие журнала; блок считается доставленным |
| повтор того же `BODID` | MES возвращает прежнее подтверждение; повторного блока не возникает |
| `ErrorMessage` с `ErrorType=Data` (неизвестная партия, экземпляр) | без автоповтора → карантин исходящих, задача администратору; **для блока** — дополнительно тревога мастеру: «блок в MES не доставлен», изделие остаётся заблокированным в `ant` |
| нет `ConfirmBOD` за время ожидания при `ConfirmationCode=Always` | повтор с тем же `BODID`, затем карантин и тревога |
| транспорт: `5xx`, таймаут | автоповтор с экспоненциальной задержкой, затем карантин |
| `401` / `403` | канал на паузу, тревога |
| исходящее не проходит XSD, неизвестная `versionID` в ответе | канал `degraded`, отправка остановлена (AD-18) |
| входящее сообщение не проходит схему | карантин приёма с кодом (FR-30); отправителю — `ConfirmBOD` с ошибкой |

## 9. Граница эмуляции (NFR-TEST-2)

- **В MVP stand-а нет.** Есть адаптер, схемы, эталонные сообщения и контрактный тест: исходящие сообщения из порта валидируются XSD MESA и сравниваются с эталонами; эталонные входящие (`ProcessOperationsSchedule`, `NotifyOperationsEvent`) через адаптер дают ожидаемые факты.
- В демо факты операций дают stand-ы источников и терминал исполнителя через edge-агент — MES для сквозного сценария не требуется.
- **В очереди** — stand MES за тем же протоколом (схема `stand_mes`): приём блоков с подтверждением, выдача заданий, сбои.
- Stand не имитирует планирование и диспетчеризацию MES.

**Где в коде:** `backend/internal/infrastructure/integration/mes/b2mml/` (клиент), `…/mes/b2mml/stand/` (stand, очередь).

## 10. Путь к реальной MES

1. Для MES, понимающей B2MML (или ESB с преобразованием), — сменить адрес канала в конфигурации.
2. Для 1С:MES — тот же порт через адаптер 1С (EnterpriseData); для ГОЛЬФСТРИМ — адаптер к XML интеграционной шины ЛОЦМАН; для самописной — преобразование B2MML ↔ её формат на её стороне или отдельным адаптером `infrastructure/integration/mes/‹система›`.
3. Если MES сама фиксирует контроль, «прямой аналог» зоны `ant` (например, модуль 8D-управления качеством) становится соседней системой, а не MES: обмен результатами контроля и несоответствиями описывается отдельным контрактом.
4. Порт MES, домен и контракт событий `ant` не меняются (кейс §3.1).

## Уточнить после появления кода

- Состав подмножества B2MML в `contracts/integrations/mes/`, имена файлов схем и эталонов, имя контрактного теста.
- Имена наших типов событий для фактов из MES и для квитанций (`contracts/events/catalog.yaml`).
- Адрес операции приёма для входящих сообщений MES в `contracts/openapi.yaml`.
- Коды ошибок `ErrorMessage`, которые stand/адаптер различают, и их связь с `contracts/errors.yaml`.
- Ключи конфигурации канала MES в `deploy/config/ant.yaml`.
- Объём и сроки stand-а MES (эпик 43).

# Интеграция с MES

Обмен с системой управления производством (MES) без повторного ручного ввода (FR-93, кейс §3.3): из MES в «Главный» приходят задания и события операций, из «Главный» в MES — блокировки изделий и партий (чтобы заблокированное изделие не получило следующую операцию), результаты контроля и задания на доработку. Контракт — нейтральный, по ISA-95 / IEC 62264 в форме B2MML. В MVP — адаптер в коде, контракт `mes.isa95.v1`, эталонные сообщения, контрактный тест и stand MES в роли `stands` (эпик 43): блок из «Главного» подтверждается MES, заблокированное изделие не получает следующую операцию.

**Опоры:** FR-90, FR-93, FR-95, FR-96; AD-7, AD-18, AD-20, AD-30; кейс §3.3, §4.3, §5.3; критерии Т2, О8. Общие правила — [README.md](README.md).

## 1. Какая MES и почему нейтральный контракт

На месте «MES» у предприятия может стоять разное (по открытым материалам вендоров):

| Система | Что известно | Интерфейс |
|---|---|---|
| 1С:ERP — пооперационное планирование («MES») и 1С:MES ОУП | задания, этапы, выполнение операций, учёт брака; обмен с 1С:ERP через EnterpriseData с квитированием | через адаптер 1С / EnterpriseData |
| Галактика MES | сменные задания, учёт исполнителем, электронный паспорт, прослеживаемость | через Галактика ESB |
| ГОЛЬФСТРИМ (АСКОН) | MES на платформе ЛОЦМАН:PLM; сопроводительные листы, наряды, акты о браке, работа ОТК | ЛОЦМАН API / интеграционная шина (XML) |
| Самописные цеховые системы | цеховой учёт в собственных разработках | нужен нейтральный контракт |

Единого российского стандарта обмена с MES нет, поэтому «Главный» говорит на международном нейтральном: **ISA-95 / IEC 62264**, сообщения — **B2MML** от MESA International (XSD, релиз V7, есть JSON-вариант, сгенерированный из XSD) (проверено по документации). Адаптер конкретной MES переводит B2MML в её формат; наш контракт `mes.isa95.v1` — **проектное предположение**: подмножество транзакций B2MML в форме B2MML-JSON (AD-18).

## 2. Модель сообщений B2MML (проверено по документации)

- Имя сообщения — `‹Глагол›‹Существительное›`: `ProcessOperationsSchedule`, `SyncMaterialLot`, `NotifyOperationsEvent`, `ProcessTestResult`.
- Обёртка: `ApplicationArea` (`Sender`, `Receiver`, `CreationDateTime`, **`BODID`** — идентификатор сообщения) + `DataArea` (глагол + существительные); на корне — `releaseID` / `versionID`.
- Подтверждения: `ConfirmBOD` (`ConfirmationCode` = `Always | Never | OnError`), ответы `Acknowledge*`; ошибки — `ErrorMessage` (`ErrorCode`, `ErrorDescription`, `ErrorType`).
- Состояние партии — `Disposition`: `Planned`, `In-Process`, **`Restricted`** («не разрешена к нормальному использованию; пример — партия ожидает решения по качеству»), `UnRestricted`, `Closed` («израсходована, продана или утилизирована»), `Other`. Наши уровни сдерживания ложатся на стандарт без натяжки.

## 3. Направления, данные и шаги процесса

| Событие | Сообщение B2MML | Направление | Подтверждение | Шаг процесса |
|---|---|---|---|---|
| задание / партия запуска выдана | `ProcessOperationsSchedule` (`OperationsRequest` → `SegmentRequirement`: деталь, количество, маршрут) | MES → «Главный» | `AcknowledgeOperationsSchedule` | запуск процесса изделий |
| операция начата / завершена / пауза | `NotifyOperationsEvent` (`Category`: начало, конец, остановка) или `SyncOperationsPerformance` | MES → «Главный» | `ConfirmBOD` при `OnError` | каждая операция маршрута |
| блок изделия или партии (защитная реакция) | `SyncMaterialSubLot` / `SyncMaterialLot` с `Disposition=Restricted` + `WorkAlert` мастеру | «Главный» → MES | `ConfirmBOD` с `Always` — блок должен быть подтверждён | на любом шаге: сигнал по карте реакций, блок партии, область риска |
| снятие блока (решение человека) | `SyncMaterialLot` `Disposition=UnRestricted` | «Главный» → MES | `ConfirmBOD` | после решения на ЗТ-Р или сужения области риска |
| результат контроля | `ProcessTestResult` (`TestableObjectID` — экземпляр или партия, `EvaluatedCriterionResult`, `PropertyMeasurement`) | «Главный» → MES | `AcknowledgeTestResult` | каждая подписанная ЗТ-1…ЗТ-6 |
| доработка / ремонт | `ProcessOperationsSchedule` с сегментом доработки и ссылкой на решение | «Главный» → MES | `AcknowledgeOperationsSchedule` | ЗТ-Р: переделка, ремонт |
| окончательный брак | `SyncMaterialSubLot` `Disposition=Closed` | «Главный» → MES | `ConfirmBOD` | ЗТ-Р: списать |
| поиск затронутых изделий | `GetMaterialLot` → `ShowMaterialLot` с `AssemblyLot` (генеалогия) | «Главный» → MES → «Главный» | ответ `Show` | область риска по партии (описание, не MVP) |
| ошибка обработки | `ConfirmBOD` с описанием / `ErrorMessage` | в обе стороны | — | — |

**Как блок согласован с моделью статусов (AD-30).** Блок — защитная реакция движка (ось «сдерживание», модуль `nonconformity`), MES только получает о нём сообщение. Если MES всё же прислала факт «операция начата» над заблокированным изделием, факт **принимается** с сигналом нарушения (внешний факт не отклоняется), затем реакция и повторный блок в MES. Команду API или терминала «Главного», нарушающую блок, гард отклоняет.

**«Без повторного ручного ввода».** Когда MES есть, факты операций приходят из неё, и исполнителю не нужно дублировать их на терминале «Главного»; когда MES нет, те же факты даёт терминал исполнителя (FR-137). История изделия одна и та же, различается только `source_kind` факта (FR-140).

## 4. Источник достоверных сведений

| Данные | Источник истины |
|---|---|
| пооперационный план, сменные задания | MES (или ERP, если MES нет) |
| факты операций: начало, конец, исполнитель, оборудование | MES; без MES — терминал исполнителя «Главного» |
| результаты контроля, сдерживание, решения по изделию | «Главный» |
| генеалогия партий в MES (`AssemblyLot`) | MES — для своих партий; генеалогия изделий — «Главный» (межизделийная стадия) |

## 5. Сопоставление идентификаторов

- Внешний ID — `ID` существительного: `OperationsRequest.ID`, `MaterialLot.ID`, `MaterialSubLot.ID`; ID сообщения — `BODID`.
- Партия MES ≠ серия ERP: связь «экземпляр «Главного» ↔ партия MES ↔ серия 1С» хранится явно — несколькими записями `reference.external_id.mapped` на один наш ID.
- В исходящих `TestableObjectID` / `MaterialSubLot.ID` адаптер ставит ID MES, если соответствие есть, иначе — естественный ключ `‹обозначение›#‹заводской номер›`, а наш ID — в `Description` / расширении.
- Входящий `BODID` становится основой `event_id` факта (детерминированно), поэтому повторная доставка того же сообщения — дубль, а не новое событие (FR-31).

## 6. Контракт `mes.isa95.v1`

- Форма — B2MML-JSON (имена элементов и атрибутов по XSD B2MML V7), `releaseID="7.01"`, `versionID="mes.isa95.v1"`.
- **Реализовано в MVP (эпик 31):** `ProcessOperationsSchedule` и `NotifyOperationsEvent` (внутрь), `SyncMaterialSubLot` и `SyncMaterialLot` (блок и снятие блока наружу), `ConfirmBOD` (подтверждения). **Описание, следующий шаг:** `AcknowledgeOperationsSchedule`, `SyncOperationsPerformance`, `ProcessTestResult` / `AcknowledgeTestResult`, задание на доработку в MES.
- `ProcessSegmentID` — код операции маршрута, тот же, что `ant:properties/@operationCode` шага BPMN (`010` — мехобработка, `030` — сварка…): по нему событие операции MES попадает на шаг процесса. `SegmentResponseID` связывает начало и конец одного выполнения (наш `operation_run_id` = `MES-‹SegmentResponseID›`).
- Транспорт в MVP — HTTP-привязка: исходящие — `POST ‹канал›/SyncMaterialSubLot`, `POST ‹канал›/SyncMaterialLot` (ответ — `ConfirmBOD`); входящие шлюз Главного читает с `GET ‹канал›/outbox` и подаёт в обычный приём (источник `mes.b2mml`, вид «внешняя система»); сверка — `GET ‹канал›/about`. Брокер — сменный адаптер того же порта (см. [observability-kafka-otel.md](../observability-kafka-otel.md)).
- Схемы — `contracts/integrations/mes/b2mml/*.schema.json` (JSON Schema подмножества; имена — как в стандарте) и `contracts/integrations/mes/binding/` (HTTP-привязка), эталоны — `contracts/integrations/mes/examples/`, описание — `contracts/integrations/mes/README.md`.
- Каждое исходящее сообщение до отправки проверяется JSON Schema подмножества (FR-111); входящие разбираются терпимо (лишние элементы стандарта отбрасываются) и проверяются схемой подмножества. Родные XSD MESA в репозиторий не копируются — подмножество сверено с ними вручную.
- `BODSuccessMessage/Duplicate` — наше расширение подтверждения: повтор того же `BODID` уже принят, второго блока нет (AD-7).

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
        "Description": "Блок по сигналу контроля; Главный: ENT01:FL-0007; ожидается решение контролёра",
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

| Ответ | Поведение «Главного» |
|---|---|
| `ConfirmBOD` с успехом / `Acknowledge*` | квитанция → событие журнала; блок считается доставленным |
| повтор того же `BODID` | MES возвращает прежнее подтверждение; повторного блока не возникает |
| `ErrorMessage` с `ErrorType=Data` (неизвестная партия, экземпляр) | без автоповтора → карантин исходящих, задача администратору; **для блока** — дополнительно тревога мастеру: «блок в MES не доставлен», изделие остаётся заблокированным в «Главный» |
| нет `ConfirmBOD` за время ожидания при `ConfirmationCode=Always` | повтор с тем же `BODID`, затем карантин и тревога |
| транспорт: `5xx`, таймаут | автоповтор с экспоненциальной задержкой, затем карантин |
| `401` / `403` | канал на паузу, тревога |
| исходящее не проходит XSD, неизвестная `versionID` в ответе | канал `degraded`, отправка остановлена (AD-18) |
| входящее сообщение не проходит схему | карантин приёма с кодом (FR-30); отправителю — `ConfirmBOD` с ошибкой |

## 9. Граница эмуляции (NFR-TEST-2)

- **Stand MES (эпик 43)** — за той же HTTP-привязкой B2MML-JSON (`/stand/mes/b2mml/`): `about`, приём `SyncMaterialSubLot` / `SyncMaterialLot` с ответом `ConfirmBOD` (повтор `BODID` — `Duplicate`, второго блока нет; нарушение схемы — `ErrorType = Contract`), `outbox` с `ProcessOperationsSchedule` и `NotifyOperationsEvent` для «Главного». Состояние экземпляров и партий: `Restricted` (`QC-HOLD`, «Изолятор ОТК») или `UnRestricted`. **Главное правило MES:** заблокированному экземпляру следующая операция не выдаётся — кнопка «Выдать операцию» отказывает, событие `Start` не формируется; после снятия блока операция выдаётся, и событие приходит в «Главный» тем же шлюзом, что у реальной MES.
- Сбои — только с пульта тестовых сценариев через служебный порт stand-ов: `offline`, `error` (`param` 4xx — `ConfirmBOD` с `ErrorType = Data`, код `SUBLOT_UNKNOWN` / `LOT_UNKNOWN`; 5xx — повтор тем же `BODID`; `match` — `hold:F-001`, `release:F-001` или подстрока ID), `delay`, `duplicate` (ответ потерян), `corrupt` (шина объявляет только `mes.isa95.v2` — канал `degraded`).
- В демо факты операций по-прежнему дают терминал исполнителя и stand-ы источников; события MES приходят только по кнопкам страницы stand-а. Stand не имитирует диспетчирование, расписание, учёт оборудования и персонала и брокер сообщений; состояние — в памяти роли `stands`.

**Где в коде:** `backend/internal/infrastructure/integration/mes/b2mml/` (клиент, порт `application/mes.Channel`), `backend/internal/domain/mes/` (перевод входящих, план блока), `backend/internal/application/mes/` (шлюз, реакция, отправка, проекции); `…/mes/b2mml/stand/` — stand, эпик 43.

## 10. Путь к реальной MES

1. Для MES, понимающей B2MML (или ESB с преобразованием), — сменить адрес канала в конфигурации.
2. Для 1С:MES — тот же порт через адаптер 1С (EnterpriseData); для ГОЛЬФСТРИМ — адаптер к XML интеграционной шины ЛОЦМАН; для самописной — преобразование B2MML ↔ её формат на её стороне или отдельным адаптером `infrastructure/integration/mes/‹система›`.
3. Если MES сама фиксирует контроль, «прямой аналог» зоны «Главного» (например, модуль 8D-управления качеством) становится соседней системой, а не MES: обмен результатами контроля и несоответствиями описывается отдельным контрактом.
4. Порт MES, домен и контракт событий «Главного» не меняются (кейс §3.1).

## В коде (эпик 31)

- **Типы событий:** задания — `mes.job.received`; события операций — те же `operation.run.started` / `paused` / `resumed` / `finished` изделия, что у терминала исполнителя (различается источник, FR-140); блок в MES — реакция `mes.hold.requested` (бизнес-ключ `‹изделие или партия›/hold|release/‹цикл›`, поток `erp_message:‹ключ›`), квитанция — `mes.hold.responded`.
- **Когда уходит блок** (`domain/mes.PlanHold`): сдерживание `item_hold` / `lot_hold` правилом или человеком, изоляция изделия, распространение по генеалогии — «заблокировать», если ещё не заблокировано; снятие — когда человек снял все записи, державшие блок, понизил уровень или принял решение по партии (кроме «не годна»). Снятие основания правилом блок не снимает (AD-27).
- **Сопоставление ID:** экземпляр, сотрудник, оборудование MES — только по `reference.external_id.mapped` (система `mes`); MES может вернуть наш ID изделия из блока. Нет соответствия или кода операции в BPMN — сообщение откладывается с причиной, факт не выдумывается (FR-123); неизвестный исполнитель — `null`.
- **Коды ошибок:** код MES из `ErrorMessage/ErrorCode` пишется в `mes.hold.responded.error_code` как есть; недоступность — `mes.unavailable` (`contracts/errors.yaml`).
- **Контрактный тест:** `go test ./internal/infrastructure/integration/mes/... ./internal/domain/mes/...` — эталоны по схемам, блок и снятие → эталонные сообщения, классы `ConfirmBOD`, шлюз входящих с BPMN фланца и схемами приёма, сквозной путь «сдерживание → `mes.hold.requested` → MES → `mes.hold.responded` → `mes.block.list`».
- **Конфигурация** (подключено в `cmd/ant`: `mes.go`, роль `outbox` под арендой `outbox.mes`, реакция блоков — в `projector`): `integrations.enabled: [mes]`, `mes.b2mml.base_url`, `logical_id`, `user`, `password_file`, `stand`, `timeout`, `poll`, `recheck`, `pull_every`, `retry_max`. Без `mes` в `integrations.enabled` реакция блоков не пишет `mes.hold.requested` — блок без канала никуда не уйдёт.
- **Stand MES** — эпик 43, раздел ниже.

## Как поднять stand MES (эпик 43)

«Главный» в профиле `demo` уже поднимает stand: `docker compose up` (или `make demo`) — в `deploy/config/ant.yaml` у профиля `demo` `integrations.enabled` содержит `mes`, `mes.b2mml.stand: true`, `base_url` пуст — адаптер идёт в `http://127.0.0.1:8491/stand/mes/b2mml` своего хоста (роль `stands`).

1. **Экран «Интеграции»** стола администратора: MES — «установлена», режим «стенд»; «Проверить соединение» — сверка `GET …/about` (релиз B2MML `7.01`, `mes.isa95.v1`).
2. **Блок ОТК уходит в MES:** сдерживание изделия → реакция `projector` пишет `mes.hold.requested` → роль `outbox` (аренда `outbox.mes`) отправляет `SyncMaterialSubLot` с `Disposition = Restricted` → stand отвечает `ConfirmBOD` → `mes.hold.responded accepted`; снятие блока — `UnRestricted`. Выключенная на экране MES — блоки ждут и уходят после включения.
3. **Страница stand-а** — `http://127.0.0.1:${ANT_STANDS_PORT:-8491}/stand/mes/`: экземпляры и партии с состоянием блока, кнопки «Выдать операцию» (заблокированному — отказ «экземпляр заблокирован ОТК»), «Завершить» (событие `End`), «Выдать задание» (`ProcessOperationsSchedule` → `mes.job.received`), журнал сообщений «Главного» с `BODID`, подтверждением, числом доставок и сбоями.
4. **Состояние для проверок** — `GET /stand/mes/state` (JSON); сбои — `POST /stand/_control/mes/faults` `{"kind":"error","param":422,"match":"hold:F-002"}` с пульта сценариев, снять — `DELETE /stand/_control/mes/faults`.

Проверка без стека: `go test ./internal/infrastructure/integration/mes/...` — адаптер на stand-е: блок и повтор, отказ в операции заблокированному, снятие и событие операции в шлюз, задания, сбои.

# bpmn-ext — расширение BPMN `urn:ant:bpmn-ext:1`

Процесс — BPMN 2.0 XML; наши свойства — в стандартных `extensionElements` пространства `urn:ant:bpmn-ext:1`, описание шага — в стандартном `documentation` (AD-17, FR-10, FR-12). Подписываемая версия — XML целиком, хеш = H(байты XML как загружены).

| Файл | Что |
|---|---|
| `ant.json` | дескриптор moddle — источник схемы; из него генерируются Go-структуры XML, XSD и таблица документации (эпик 02); его же подключают bpmn-js и панель свойств |
| `rules.yaml` | что обязательно, допустимые значения, поддерживаемое подмножество BPMN, переменные языка условий |
| `norm-anchors.yaml` | проверенный перечень нормативных опор для `ant:normRef` (FR-156, PRD §11.17) |

## Элементы

Все элементы лежат прямо в `bpmn:extensionElements` узла:

| Элемент | Сколько | Свойства (FR-12) |
|---|---|---|
| `ant:properties` | ровно один у узла; у дорожки — цех и склад | `stepKey` (обязателен у каждого узла), `stepKind`, `workshop`, `warehouse`, `specialProcess` (FR-151), `bufferPlace`, `closesZoneAccess` (FR-20), `paperAttester` (AD-43), `closingPoint`, `inspectionPoint`, `operationCode`, `reworkLimit` + `reworkLimitScope` (FR-18), `erpAction`, `triggerEventType` (Д-5), `timerScope` (Д-8), `outcome` |
| `ant:inspection` | 0–1 | метод контроля, фаза, покрытие видов дефектов (FR-14), карта контроля, порог качества наблюдения |
| `ant:requirement` | 0–n | требование КД: характеристика, допуск, документ и ревизия |
| `ant:zoneRef` | 0–n | зона изделия |
| `ant:presentationPoint` | 0–1 | точка предъявления: полномочие, роль, вид клейма, срок ожидания, полномочие при повторном предъявлении, приёмка ПЗ/ВП (FR-19) |
| `ant:precondition` | 0–n | предусловие на дату операции: вид, режим `block` / `record_violation`, параметр (FR-17) |
| `ant:document` | 0–n | документ шага — шаблон нормативного слоя (AD-12; шаблоны заводит эпик 28) |
| `ant:reactionMap` | 0–1 | карта реакций `‹id›@‹версия›` (FR-48) |
| `ant:norm` | 0–1 | норма времени, ожидания, пропускной способности, порог простоя (FR-5) |
| `ant:normRef` | 0–n | нормативная опора: `standard`, `clause`, `<ant:check>`, `<ant:systemAction>` (FR-156) |

`stepKey` стабилен между версиями; смена смысла — новый ключ. Счётчики карты, заготовки и изделия прежних версий привязываются к `stepKey` (AD-17). Шлюзы явного слияния получают ключ `‹ключ узла›.merge`.

## Поддерживаемое подмножество и решения по стартовому процессу

- Задачи трёх видов (`stepKind`): операция, автоматизированный контроль, контроль человеком; плюс перемещение и хранение (FR-11, FR-16).
- Шлюзы: исключающий, параллельный, включающий (развилка и слияние).
- **terminate** — поддерживается: завершает все токены изделия (Д-4).
- **Старт по сообщению** — поддерживается: `startEvent` с `messageEventDefinition` и `ant:properties/@triggerEventType` = тип события каталога (для фланца — `erp.order.received`) (Д-5).
- **Неявные слияния запрещены** (Д-6): у любого узла, кроме шлюза, — не больше одной входящей стрелки; загрузчик отклоняет нарушение кодом `process.implicit_merge` с id узла. В копии стартового процесса все 19 неявных слияний заменены явными исключающими шлюзами `J_‹узел›`.
- **Граничный таймер** — окно времени на шаге (Д-8): `boundaryEvent` + `timerEventDefinition/timeDuration` (ISO 8601) + `timerScope`: `until_started` — окно закрывается началом операции (окно «кромки → сварка ≤ 8 ч»), `activity` — всё время шага. Истечение — переход по ветке таймера.
- Вызываемый подпроцесс (`callActivity`) — стек вызовов в состоянии токена с возвратом в точку вызова (AD-17); итог — `ant:properties/@outcome` достигнутого конечного события, доступный условиям как `nc.outcome`.

## Язык условий `urn:ant:expr:1` (Д-7)

Условие на стрелке — `bpmn:conditionExpression` (`xsi:type="bpmn:tFormalExpression"`); язык задаётся атрибутом `expressionLanguage="urn:ant:expr:1"` у `bpmn:definitions`. Язык минимальный и детерминированный: нет вызовов функций, арифметики, чисел с плавающей точкой и обращения к часам (AD-4).

```ebnf
expr        = or_expr ;
or_expr     = and_expr , { "or" , and_expr } ;
and_expr    = not_expr , { "and" , not_expr } ;
not_expr    = "not" , not_expr | primary ;
primary     = "(" , expr , ")" | comparison ;
comparison  = field , op , literal ;
op          = "==" | "!=" | "<" | "<=" | ">" | ">=" ;
field       = ident , { "." , ident } ;
ident       = letter_lc , { letter_lc | digit | "_" } ;
literal     = string | integer | "true" | "false" ;
string      = "'" , { any_char - "'" } , "'" ;
integer     = [ "-" ] , digit , { digit } ;
```

- Приоритет: `not` > `and` > `or`; скобки меняют порядок. Пробелы между лексемами — любые.
- Поле слева, литерал справа. Поля — только из состояния изделия и решения: перечень и типы — `rules.yaml` → `conditions.variables`.
- Типы: `enum` — строковый литерал из перечня, операции `==` и `!=`; `boolean` — `true`/`false`, операции `==` и `!=`; `integer` — целый литерал в пределах ±(2^53−1), любые операции сравнения (сравнения по величине — только над целыми).
- Литералы `true` и `false` — для булевых полей (`tools.accounted == true`); это уточнение эпика 00 к решению Д-7.
- Нарушение — `process.condition_invalid` с id стрелки.

Примеры, которые проходят: `decision == 'accept'`, `decision == 'accept' or decision == 'accept_with_concession'`, `nc.outcome == 'scrapped' or nc.outcome == 'returned'`, `not (test.result == 'tight')`, `rework.count < 3 and disposition == 'rework'`. В копии стартового процесса `||` записано как `or`.

## Проверка

`contracts/scripts/check-bpmn.mjs` для каждого `normative/process/*.bpmn`: разбор bpmn-moddle с этим дескриптором — ноль предупреждений (то же, что показывает bpmn-js при импорте); у каждого узла DI-фигура, у каждой стрелки DI-линия; только поддерживаемые элементы; нет неявных слияний; у каждого узла `stepKey` по шаблону, уникальный; значения по `rules.yaml`; ссылки на классификатор, зоны, политику, каталог событий существуют; опоры — из `norm-anchors.yaml`; условия — по грамматике; список шагов `*.steps.yaml` совпадает с BPMN.

// СГЕНЕРИРОВАНО contracts/scripts/gen-ts.mjs (make generate) — руками не править (AD-20).
// Источник: contracts/events/common/*.json, contracts/events/‹семейство›/*.v‹N›.json


/**
 * Учётная запись активирована — заявка на регистрацию активирована администратором с назначением роли (FR-128).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AccessAccountActivatedV1".
 */
export interface AccessAccountActivatedV1 {
/**
 * Сотрудник.
 */
person_id: string
/**
 * Логин.
 */
login: string
/**
 * Роль при активации.
 */
initial_role_id?: string
}
/**
 * Назначение на пост снято — сотрудник снят с поста в смене.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AccessAssignmentClearedV1".
 */
export interface AccessAssignmentClearedV1 {
/**
 * Сотрудник.
 */
person_id: string
/**
 * Пост.
 */
workplace_id: string
/**
 * Смена.
 */
shift_id: string
reason?: Reason
}
/**
 * Причина: код и текст.
 */
export interface Reason {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Назначение на пост — исполнителя назначает мастер (только допущенного по квалификации); контролёра — по документу «запрос мастера → согласование начальника ОТК» (PRD §11.18).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AccessAssignmentSetV1".
 */
export interface AccessAssignmentSetV1 {
/**
 * Сотрудник.
 */
person_id: string
/**
 * Пост (рабочее место).
 */
workplace_id: string
/**
 * Смена.
 */
shift_id: string
/**
 * Кого назначают.
 */
assignee_role: ("performer" | "quality_inspector")
/**
 * Документ согласования начальника ОТК — для контролёра обязателен.
 */
approval_document_id?: string
}
/**
 * Сотрудник заведён — сотрудник с условным идентификатором (псевдонимом); соответствие человеку хранится отдельно в access (FR-78).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AccessPersonRegisteredV1".
 */
export interface AccessPersonRegisteredV1 {
/**
 * Псевдоним сотрудника.
 */
person_id: string
/**
 * Отображаемое имя (условное).
 */
display_name: string
/**
 * Подразделение.
 */
org_unit?: string
}
/**
 * Квалификация или аттестация выдана — квалификация с областью и сроком; проверяется на дату операции (FR-80, FR-17).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AccessQualificationGrantedV1".
 */
export interface AccessQualificationGrantedV1 {
/**
 * Сотрудник.
 */
person_id: string
/**
 * Вид квалификации (например, аттестация сварщика по способу, материалу, толщине).
 */
qualification_id: string
/**
 * Область действия: путь `здание/цех/участок/рабочее место` (иерархия областей, AD-15).
 */
scope?: string
/**
 * Номер удостоверения.
 */
certificate_ref?: string
/**
 * Начало действия (доменное время).
 */
valid_from: string
/**
 * Окончание действия; отсутствует — бессрочно.
 */
valid_until?: string
}
/**
 * Квалификация отозвана — квалификация больше не действует с указанной даты (FR-80).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AccessQualificationRevokedV1".
 */
export interface AccessQualificationRevokedV1 {
/**
 * Сотрудник.
 */
person_id: string
/**
 * Вид квалификации.
 */
qualification_id: string
/**
 * С какого момента не действует.
 */
effective_from: string
reason?: Reason1
}
/**
 * Причина: код и текст.
 */
export interface Reason1 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Токен вставлен или извлечён — агент токена сообщает о вставке или извлечении токена; вынутый ключ меняет статус поста не позднее 2 с (FR-6, FR-84).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AccessTokenPresenceChangedV1".
 */
export interface AccessTokenPresenceChangedV1 {
/**
 * Владелец токена.
 */
person_id: string
/**
 * Рабочее место.
 */
workplace_id: string
/**
 * Токен вставлен.
 */
present: boolean
}
/**
 * Допуск к рабочему месту открыт — правило домена выполнено: СКУД в зоне ∧ роль в области места ∧ квалификация на дату ∧ назначение на пост ∧ токен и PIN (AD-15, FR-83).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AccessWorkplaceAdmittedV1".
 */
export interface AccessWorkplaceAdmittedV1 {
/**
 * Рабочее место.
 */
workplace_id: string
/**
 * Сеанс рабочего места.
 */
workplace_session_id: string
/**
 * Сотрудник.
 */
person_id: string
/**
 * Смена.
 */
shift_id?: string
/**
 * Факты, на которых основан допуск (СКУД, назначение, квалификация).
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
checks_event_ids?: string[]
}
/**
 * Сеанс рабочего места завершён сотрудником — сотрудник закрыл сеанс рабочего места.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AccessWorkplaceReleasedV1".
 */
export interface AccessWorkplaceReleasedV1 {
/**
 * Рабочее место.
 */
workplace_id: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
workplace_session_id: string
}
/**
 * Допуск к рабочему месту снят автоматически — извлечён токен, сотрудник вышел из зоны, закончилась смена или отозвана квалификация (AD-15).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AccessWorkplaceRevokedV1".
 */
export interface AccessWorkplaceRevokedV1 {
/**
 * Рабочее место.
 */
workplace_id: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
workplace_session_id: string
/**
 * Причина.
 */
cause: ("token_removed" | "zone_exit" | "shift_ended" | "qualification_revoked" | "assignment_cleared" | "policy_changed")
}
/**
 * Проход через точку СКУД — событие прохода в зону или из зоны через порт СКУД (FR-82).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AccessZonePassedV1".
 */
export interface AccessZonePassedV1 {
/**
 * Сотрудник.
 */
person_id: string
/**
 * Зона доступа.
 */
zone_id: string
/**
 * Направление.
 */
direction: ("enter" | "exit")
/**
 * Считыватель.
 */
reader_id?: string
}
/**
 * Проверка анализатора — экзамен, прогон эталонного набора, сравнение с людьми и прошлой версией, теневой режим; ориентиры AIAG: пропуск брака < 2 %, ложные тревоги < 5 % (FR-101).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AnalyzerCheckRecordedV1".
 */
export interface AnalyzerCheckRecordedV1 {
/**
 * Паспорт допуска.
 */
passport_id: string
/**
 * Вид проверки.
 */
check_kind: ("exam" | "reference_set" | "shadow_comparison" | "drift_monitor")
/**
 * Доля пропуска брака.
 */
escape_rate_bp?: number
/**
 * Доля ложных тревог.
 */
false_alarm_rate_bp?: number
/**
 * Доля расхождений с людьми.
 */
disagreement_rate_bp?: number
/**
 * Критерии выполнены.
 */
passed: boolean
versions: AnalyzerVersions
}
/**
 * Вектор версий.
 */
export interface AnalyzerVersions {
/**
 * Ревизия изделия (КД).
 */
item_revision?: string
/**
 * Карта контроля с версией: `‹id›@‹версия›`.
 */
recipe_ref: string
/**
 * Конфигурация камеры и света.
 */
camera_config?: string
/**
 * Калибровка.
 */
calibration?: string
/**
 * Версия анализатора (внешнего модуля).
 */
analyzer_version: string
/**
 * Профиль порогов.
 */
threshold_profile?: string
/**
 * Версия контракта данных источника.
 */
contract_version: string
/**
 * Версия приложения источника.
 */
app_version?: string
}
/**
 * Паспорт допуска введён в действие — после маршрута подписей (ОТК, технолог, метролог; при изменении ТП/КД — ВП и держатель КД): уровень доверия 0–4, разрешённые и запрещённые автоматические действия; допущенная версия заморожена (FR-98, FR-101, AD-29).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AnalyzerPassportAdmittedV1".
 */
export interface AnalyzerPassportAdmittedV1 {
/**
 * Паспорт.
 */
passport_id: string
/**
 * Стадия.
 */
stage: ("shadow" | "pilot" | "active")
/**
 * Уровень доверия (contracts/analyzer-trust-levels.yaml).
 */
trust_level: number
/**
 * Карта контроля `‹id›@‹версия›`.
 */
recipe_ref: string
versions: AnalyzerVersions1
/**
 * Предыдущий допущенный паспорт — для отката.
 */
previous_passport_id?: string
/**
 * Протокол допуска.
 */
document_id: string
/**
 * Анализатор (внешняя система видеофиксации и её модель), к которому относится паспорт: например, `vqc-weld`.
 */
analyzer_id?: string
/**
 * Вид анализатора: визуальный контроль (VisionQC) или контроль действий оператора (OperatorVision).
 */
analyzer_kind?: ("visionqc" | "operatorvision")
/**
 * Название анализатора для людей, например «Визуальный контроль шва (КТ-3)».
 */
title?: string
}
/**
 * Допущенная конфигурация контура.
 */
export interface AnalyzerVersions1 {
/**
 * Ревизия изделия (КД).
 */
item_revision?: string
/**
 * Карта контроля с версией: `‹id›@‹версия›`.
 */
recipe_ref: string
/**
 * Конфигурация камеры и света.
 */
camera_config?: string
/**
 * Калибровка.
 */
calibration?: string
/**
 * Версия анализатора (внешнего модуля).
 */
analyzer_version: string
/**
 * Профиль порогов.
 */
threshold_profile?: string
/**
 * Версия контракта данных источника.
 */
contract_version: string
/**
 * Версия приложения источника.
 */
app_version?: string
}
/**
 * Анализатор возвращён в работу — разрешающее действие: только начальник ОТК (FR-101, AD-27).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AnalyzerPassportReinstatedV1".
 */
export interface AnalyzerPassportReinstatedV1 {
/**
 * Паспорт.
 */
passport_id: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
suspension_event_id: string
reason: Reason2
}
/**
 * Причина: код и текст.
 */
export interface Reason2 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Паспорт допуска выведен — паспорт больше не применяется.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AnalyzerPassportRetiredV1".
 */
export interface AnalyzerPassportRetiredV1 {
/**
 * Паспорт.
 */
passport_id: string
reason: Reason3
}
/**
 * Причина: код и текст.
 */
export interface Reason3 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Паспорт допуска приостановлен — откат в сторону строгости по триггеру (дрейф, провал эталонного набора, рост расхождений, пропуск брака) — делегированное правило режима 2; действует на будущее, сильнее закреплённой версии (FR-101, AD-17, AD-29).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AnalyzerPassportSuspendedV1".
 */
export interface AnalyzerPassportSuspendedV1 {
/**
 * Паспорт.
 */
passport_id: string
/**
 * Триггер.
 */
trigger: ("drift" | "reference_set_failed" | "disagreement_growth" | "escape_detected")
/**
 * Что действует вместо.
 */
fallback: ("previous_passport" | "manual_control")
/**
 * Предыдущий допущенный паспорт.
 */
fallback_passport_id?: string
/**
 * Основания — `event_id` записей, на которые опирается запись.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
basis: string[]
/**
 * Пояснение по-русски: что увидело правило автоотката (например, «качество кадра ниже 0,70 в трёх наблюдениях подряд»).
 */
note?: string
}
/**
 * Привязка задана человеком — событие «деталь не опознана» или перепутанную деталь привязали вручную; выводы пересчитываются у обоих изделий (FR-34, AD-41).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "BindingLinkAssignedV1".
 */
export interface BindingLinkAssignedV1 {
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
subject_event_id: string
/**
 * Изделие.
 */
item_id: string
/**
 * Прежняя привязка, если была.
 */
previous_item_id?: string
/**
 * Способ.
 */
method: ("scan" | "select_expected" | "manual_entry")
reason: Reason4
}
/**
 * Причина: код и текст.
 */
export interface Reason4 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Событие привязано к изделию — межизделийная стадия разрешила носитель или контекст: одно изделие или кандидаты; надёжность привязки — от носителя (AD-41, FR-34).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "BindingLinkResolvedV1".
 */
export interface BindingLinkResolvedV1 {
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
subject_event_id: string
/**
 * Изделие (если однозначно).
 */
item_id?: string
/**
 * Кандидаты при неоднозначности.
 * 
 * Items: Кандидат.
 */
candidates?: string[]
/**
 * Как привязано.
 */
binding_basis: ("carrier" | "post_context" | "time_window" | "manual")
/**
 * Надёжность привязки.
 */
binding_reliability: ("unique" | "probable" | "ambiguous" | "unidentified")
/**
 * Носитель события `‹тип›:‹значение›`, по которому разрешалась привязка (AD-41).
 */
carrier_ref?: string
/**
 * Копия привязываемого события без изделия (тип, время, источник, data): свёртка изделия видит содержимое события в своём потоке (AD-5, AD-41).
 */
subject?: {
/**
 * Тип события каталога.
 */
event_type: string
/**
 * Версия схемы data.
 */
schema_version?: number
/**
 * Время возникновения события.
 */
occurred_at: string
/**
 * Источник события.
 */
source_id?: string
/**
 * Вид источника (FR-140).
 */
source_kind?: string
/**
 * data события как в журнале.
 */
data: {

}
}
/**
 * Изделие прежней привязки при перепривязке: событие у него больше не учитывается (AD-41).
 */
previous_item_id?: string
}
/**
 * Условная сборка импортирована — структура сборки из файла КОМПАС (идентификатор, версия, компоненты, количество, связи; геометрии нет — явно); связи переводятся в зоны и ограничения нормативного слоя: шов — объект учёта лимита ремонтов, болтовое соединение «закрывает доступ к зоне», уплотнение — порядок установки (FR-94, PRD §11.15). Схема файла — `contracts/integrations/cad/assembly.schema.json` (эпик 31). Дерево компонентов — черновик справочника типов изделий: действующим его делает утверждение человеком (AD-31, FR-23); соответствия обозначений КД нашим типам — отдельные записи reference.external_id.mapped (FR-95).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "CadAssemblyImportedV1".
 */
export interface CadAssemblyImportedV1 {
/**
 * Обозначение сборки (например, ФЛ-100.00.000 СБ).
 */
assembly_designation: string
/**
 * Версия (литера).
 */
revision: string
/**
 * Отпечаток импортированного файла.
 */
file_digest: string
/**
 * Компоненты.
 * 
 * @minItems 1
 */
components: [CadComponent, ...(CadComponent)[]]
/**
 * Связи.
 */
links?: CadLink[]
/**
 * Геометрия в файле есть; в условной сборке — false (кейс §5.3).
 */
geometry_present: boolean
/**
 * Наш тип изделия сборки по обозначению КД (FL-100.00.000).
 */
assembly_item_type_id?: string
/**
 * Наименование сборки.
 */
assembly_name?: string
/**
 * Литера КД (например, О1).
 */
lifecycle_letter?: string
/**
 * Версия формата файла условной сборки.
 */
format_version?: string
/**
 * Система-источник состава (КОМПАС-3D, эмуляция файлом; COM API7; ЛОЦМАН:PLM).
 */
source_system?: string
/**
 * Когда состав выгружен из САПР (по данным файла).
 */
exported_at?: string
/**
 * Пояснение отсутствия геометрии из файла (кейс §5.3).
 */
geometry_note?: string
/**
 * Правило сопоставления ID с учётной системой из файла (для людей).
 */
mapping_rule?: string
/**
 * Зоны изделия из связей сборки (FR-46): к ним привязываются контроль, доработки, скрытые работы и вмешательства.
 */
zones?: CadZone[]
/**
 * Ограничения нормативного слоя из связей сборки: лимит ремонтов шва (FR-18), «закрывает доступ к зоне» (FR-20), порядок установки (FR-17), крепёж и момент затяжки.
 */
constraints?: CadConstraint[]
/**
 * Важные характеристики с методами контроля (FR-14).
 */
characteristics?: CadCharacteristic[]
/**
 * Расхождения с номенклатурой учётной системы: позиции без соответствия — отчёт администратору и технологу, без автосоздания (FR-95).
 */
discrepancies?: CadDiscrepancy[]
}
/**
 * Компонент.
 */
export interface CadComponent {
/**
 * Позиция.
 */
position: string
/**
 * Обозначение.
 */
designation: string
/**
 * Количество.
 */
quantity: number
/**
 * Учитывается партией.
 */
lot_tracked: boolean
/**
 * Наш тип изделия позиции по обозначению КД.
 */
item_type_id?: string
/**
 * Наименование позиции.
 */
name?: string
/**
 * Вид позиции: деталь, сборочная единица, стандартное изделие, крепёж, покупное оборудование.
 */
kind?: ("detail" | "subassembly" | "standard_part" | "fastener" | "purchased_equipment" | "other")
/**
 * Родитель в дереве состава (тип изделия).
 */
parent_item_type_id?: string
/**
 * Единица измерения.
 */
unit?: string
/**
 * Количество на одну сборку верхнего уровня (с учётом количеств родителей).
 */
total_quantity?: number
/**
 * Учёт: по заводским номерам, по партиям, без индивидуального учёта.
 */
tracking?: ("serial" | "lot" | "none")
/**
 * Изготавливаем или покупаем.
 */
make_or_buy?: ("make" | "buy")
/**
 * Учитывается срок хранения (предусловие «партия не просрочена», FR-17).
 */
shelf_life_tracked?: boolean
/**
 * Материал.
 */
material?: string
}
/**
 * Связь сборки.
 */
export interface CadLink {
/**
 * Идентификатор связи (W-1, J-1, S-1).
 */
link_id: string
/**
 * Вид связи.
 */
kind: ("weld" | "bolted_joint" | "seal" | "other")
/**
 * Связанные позиции.
 * 
 * @minItems 1
 * 
 * Items: Позиция.
 */
components: [string, ...(string)[]]
/**
 * Вид связи в файле, если он шире перечисления kind (например, threaded_joint → other).
 */
link_type?: string
/**
 * Зона изделия, которую даёт связь.
 */
zone_id?: string
/**
 * Примечание из файла.
 */
note?: string
}
/**
 * Зона изделия.
 */
export interface CadZone {
/**
 * Зона (совпадает с идентификатором связи: W-1, J-1, S-1).
 */
zone_id: string
/**
 * Название зоны.
 */
name: string
/**
 * Вид зоны (как в справочнике типов изделий).
 */
kind: ("weld_section" | "joint" | "hole" | "surface" | "cavity" | "groove" | "other")
/**
 * Связь сборки, из которой получена зона.
 */
link_id: string
/**
 * Тип изделия, на котором лежит зона.
 */
item_type_id?: string
}
/**
 * Ограничение нормативного слоя.
 */
export interface CadConstraint {
/**
 * Идентификатор ограничения (‹связь›/‹вид›).
 */
constraint_id: string
/**
 * Вид: лимит ремонтов зоны, закрывает доступ к зонам, порядок установки, крепёж, момент затяжки.
 */
kind: ("rework_limit" | "closes_access" | "install_order" | "fastening" | "torque")
/**
 * Связь сборки.
 */
link_id: string
/**
 * Зона, к которой относится ограничение.
 */
zone_id?: string
/**
 * Лимит ремонтов зоны (для rework_limit); отсутствует — лимит не задан ни файлом, ни ТП.
 */
limit?: number
/**
 * Откуда лимит: файл сборки или шаг ТП в нормативном слое (step_key).
 */
limit_source?: string
/**
 * Зоны, к которым соединение закрывает доступ: их проверка — до закрытия (скрытые работы).
 * 
 * Items: Зона.
 */
closes_zone_ids?: string[]
/**
 * Что устанавливается первым (для install_order).
 */
first_item_type_id?: string
/**
 * Что устанавливается после (для install_order).
 */
then_item_type_id?: string
/**
 * Шаг процесса, на котором ограничение исполняется (по нормативному слою), если найден.
 */
step_key?: string
/**
 * Число крепежа (для fastening).
 */
quantity?: number
/**
 * Значение (для torque — момент, целое в единицах unit).
 */
value?: number
/**
 * Единица значения (Н·м).
 */
unit?: string
/**
 * Допуск, %.
 */
tolerance_pct?: number
/**
 * Пояснение для людей.
 */
note?: string
}
/**
 * Важная характеристика.
 */
export interface CadCharacteristic {
/**
 * Идентификатор (CC-1…).
 */
characteristic_id: string
/**
 * Название.
 */
name: string
/**
 * Методы контроля.
 * 
 * Items: Метод контроля.
 */
methods?: string[]
/**
 * Примечание.
 */
note?: string
}
/**
 * Расхождение.
 */
export interface CadDiscrepancy {
/**
 * Наш тип изделия позиции.
 */
item_type_id: string
/**
 * Обозначение КД.
 */
designation: string
/**
 * Учётная система.
 */
system: ("onec" | "galaktika" | "other")
/**
 * Нет в номенклатуре / сверка не выполнялась (номенклатура ещё не получена).
 */
reason: ("not_in_erp" | "not_checked")
/**
 * Пояснение.
 */
note?: string
}
/**
 * Общие определения полей событий v1: идентификаторы, время, числа без float, единицы, источник, изделие и носитель, дефект, материалы, вектор версий. Диалект AD-20.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "AntDefsV1".
 */
export interface AntDefsV1 {
[k: string]: unknown | undefined
}
/**
 * Измерение без float: значение = value × 10^(−scale) в единице unit.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "measurement".
 */
export interface Measurement {
/**
 * Целая мантисса значения.
 */
value: number
/**
 * Число знаков после запятой (масштаб).
 */
scale: number
/**
 * Единица измерения, код UCUM (например, `A`, `V`, `mm`, `N.m`, `Pa`, `min`).
 */
unit: string
}
/**
 * Допуск: номинал и границы; отсутствующая граница — не ограничена.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "tolerance".
 */
export interface Tolerance {
nominal?: Measurement
lower?: Measurement
upper?: Measurement
}
/**
 * Длительность по соглашению «Длительности»: значение, единица, смысл интервала и происхождение (кейс §4.6, FR-88).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "duration".
 */
export interface Duration {
/**
 * Значение длительности.
 */
value: number
/**
 * Единица длительности.
 */
unit: ("ms" | "s" | "min" | "h")
/**
 * Смысл интервала: активная обработка / полное время на участке / иной интервал.
 */
meaning: ("active_processing" | "time_at_station" | "other")
/**
 * Пояснение, если смысл — `other`.
 */
meaning_note?: string
/**
 * Происхождение: передано источником / вычислено системой.
 */
origin: ("source_reported" | "system_computed")
}
/**
 * Ссылка источника на изделие через носитель (AD-16, AD-41). Внутренний `item_id` из метки не выводится; разрешение носителя делает приём.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "item_ref".
 */
export interface ItemRef {
/**
 * Тип носителя идентификатора (AD-16): DPM DataMatrix / бирка с QR / тара и ячейка / сопроводительная карта / контекст поста / ручной ввод / внутренний ID системы.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "carrier_type".
 */
carrier_type: ("dpm_datamatrix" | "tag_qr" | "container_cell" | "route_card" | "post_context" | "manual_entry" | "internal_id")
/**
 * Значение носителя (код DataMatrix, QR, номер тары и ячейки и т. п.).
 */
value: string
/**
 * Уровень идентификации объекта (кейс §4.6, FR-34): однозначно / вероятно / неоднозначно / не опознан.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "identification_level".
 */
identification_level: ("unique" | "probable" | "ambiguous" | "unidentified")
}
/**
 * Признак дефекта в результате контроля (FR-27, FR-37). Физический дефект связывается по ключу (изделие, зона, место) без вида.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "defect".
 */
export interface Defect {
/**
 * Код вида дефекта из классификатора (FR-125); неизвестный код принимается с флагом «неизвестный вид».
 */
defect_type_code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
description?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "object_id".
 */
component_ref?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "object_id".
 */
zone_id: string
/**
 * Место в зоне (координаты, участок шва), свободная форма источника.
 */
location?: string
/**
 * Класс тяжести дефекта по ГОСТ 15467: критический / значительный / малозначительный / неизвестен.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "severity".
 */
severity: ("critical" | "major" | "minor" | "unknown")
size?: Measurement
/**
 * Размер оценён «примерно», а не измерен.
 */
size_is_estimated?: boolean
measured?: Measurement
tolerance?: Tolerance
/**
 * Доля в базисных пунктах: 0…10000 (10000 = 1,0). Float в контрактах запрещён (AD-4).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "bp".
 */
stage_confidence_bp?: number
}
/**
 * Ссылка на материал (FR-102, кейс §5.4): адрес содержимого в хранилище материалов и метаданные. Иллюстрация помечается явно.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "evidence_ref".
 */
export interface EvidenceRef {
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "digest".
 */
material_address: string
/**
 * MIME-тип материала.
 */
media_type: string
/**
 * Вид материала.
 */
kind: ("photo" | "video" | "illustration" | "protocol" | "log_excerpt" | "scan" | "other")
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "timestamp".
 */
captured_at?: string
/**
 * Иллюстрация из открытого набора, не относится к изделию (FR-102).
 */
is_illustration: boolean
/**
 * Источник, лицензия, автор — для иллюстраций обязательно.
 */
source_note?: string
}
/**
 * Вектор версий наблюдения (AD-29, FR-98): все составляющие контура контроля. Версия контракта ≠ версия анализатора ≠ версия приложения (кейс §4.7).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "analyzer_versions".
 */
export interface AnalyzerVersions2 {
/**
 * Ревизия изделия (КД).
 */
item_revision?: string
/**
 * Карта контроля с версией: `‹id›@‹версия›`.
 */
recipe_ref: string
/**
 * Конфигурация камеры и света.
 */
camera_config?: string
/**
 * Калибровка.
 */
calibration?: string
/**
 * Версия анализатора (внешнего модуля).
 */
analyzer_version: string
/**
 * Профиль порогов.
 */
threshold_profile?: string
/**
 * Версия контракта данных источника.
 */
contract_version: string
/**
 * Версия приложения источника.
 */
app_version?: string
}
/**
 * Ступень анализатора (FR-38): например, локализация → классификация; версия и уверенность у каждой.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "analyzer_stage".
 */
export interface AnalyzerStage {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
stage: string
/**
 * Версия ступени.
 */
version: string
/**
 * Доля в базисных пунктах: 0…10000 (10000 = 1,0). Float в контрактах запрещён (AD-4).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "bp".
 */
confidence_bp?: number
/**
 * Краткий вывод ступени.
 */
output_note?: string
}
/**
 * Результат измерения характеристики: значение против допуска (FR-36).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "measurement_result".
 */
export interface MeasurementResult {
/**
 * Характеристика (позиция по чертежу).
 */
characteristic: string
value?: Measurement
tolerance: Tolerance
/**
 * Оценка источника: в допуске / вне допуска / не измерено.
 */
verdict: ("within" | "outside" | "not_measured")
}
/**
 * Причина действия: код и текст.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "reason".
 */
export interface Reason5 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Конверт события v1 (кейс §4.4, FR-27, стиль CloudEvents): общий для всех типов записей — фактов источников, реакций движка, решений людей и служебных. Подписывается целиком (DSSE над каноническим JSON, AD-10). Время поступления и записи (`received_at`, `recorded_at`) ставит только ядро — в записи журнала, не здесь. Схема открыта: новые необязательные поля принимаются и сохраняются (FR-29).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "EventEnvelopeV1".
 */
export interface EventEnvelopeV1 {
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
event_id: string
/**
 * Тип события: семейство.сущность.действие — три сегмента латиницей (AD-40).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "event_type".
 */
event_type: string
/**
 * Мажорная версия схемы `data` для этого типа; несовместимое изменение — только новая мажорная версия (AD-20, FR-29).
 */
schema_version: number
/**
 * Идентификатор источника событий: устройство, шлюз, терминал, партнёр (`partner:‹код›`); в прогоне сценария — `‹run_id›/‹источник›` (AD-38).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "source_id".
 */
source_id: string
/**
 * Порядковый номер у источника (на `source_id`): обязателен для устройств, шлюзов и агентов токена; по нему верификатор проверяет непрерывность (AD-7, AD-9).
 */
source_seq?: number
/**
 * Вид источника факта (AD-2, FR-140): ручной ввод / станок / датчик / камера / внешняя система / импорт.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "source_kind".
 */
source_kind?: ("manual_entry" | "machine" | "sensor" | "camera" | "external_system" | "import")
/**
 * Надёжность факта по источнику (FR-140): высокая / средняя / низкая / неизвестна.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "reliability".
 */
reliability?: ("high" | "medium" | "low" | "unknown")
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "timestamp".
 */
occurred_at: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
correlation_id: string
/**
 * Непосредственная причина — `event_id` записи, вызвавшей эту; null — корневое событие.
 */
causation_id: (string | null)
/**
 * Идентификатор прогона сценария (AD-38); в профиле prod отсутствует.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "run_id".
 */
run_id?: string
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "item_id".
 */
item_id?: string
item_ref?: ItemRef
corrects?: Corrects
reaction?: ReactionMeta
command?: CommandMeta
integrity: Integrity
/**
 * Содержимое, специфичное для типа: схема `contracts/events/‹семейство›/‹event_type›.v‹schema_version›.json` (полиморфизм через `event_type`, AD-20).
 */
data: {

}
}
/**
 * Исправление ранее записанного события новой записью (FR-122, AD-2): было — исправляемая запись, стало — эта запись, кто — подписант, причина — здесь. Любая запись с `corrects` — критическое действие группы «защищённые данные».
 */
export interface Corrects {
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
event_id: string
reason: Reason5
}
/**
 * Метаданные реакции движка (AD-3); обязательны для записей вида «реакция».
 */
export interface ReactionMeta {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "object_id".
 */
rule_id: string
/**
 * Ревизия нормативного слоя, в которой действует правило (хеш версии процесса или номер).
 */
rule_rev?: string
/**
 * Режим автоматизации правила 1–5 (FR-50).
 */
automation_mode: number
slot: ReactionSlot
/**
 * Версия слота: 1 — первая, пересмотр — следующая.
 */
version: number
/**
 * `event_id` заменяемой версии слота; null у первой версии.
 */
supersedes: (string | null)
/**
 * Запись, из-за которой вывод пересмотрен («пересмотрен из-за события ‹id›», FR-32); null у первой версии.
 */
revised_due_to?: (string | null)
/**
 * Причины вывода — отсортированный список `event_id` записей, влияющих на вывод (вычисляет доменная функция правила).
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
causes: string[]
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "seq".
 */
basis_seq: number
}
/**
 * Слот реакции: правило, субъект, ключ срабатывания. `event_id` реакции = UUIDv5(NS_ANT, слот ‖ версия).
 */
export interface ReactionSlot {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "object_id".
 */
rule_id: string
/**
 * Поток записи (AD-39): `item:‹item_id›`, `‹вид_объекта›:‹id›` или `global`.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "stream_ref".
 */
subject: string
/**
 * Ключ срабатывания внутри субъекта.
 */
trigger_key: string
}
/**
 * Метаданные команды человека (AD-39, AD-14); обязательны для записей вида «решение».
 */
export interface CommandMeta {
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
command_id: string
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "seq".
 */
basis_seq: number
/**
 * Потоки, которые проверил гард команды: после `basis_seq` в них не должно быть новых записей `guard_relevant`.
 * 
 * Items: Поток записи (AD-39): `item:‹item_id›`, `‹вид_объекта›:‹id›` или `global`.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "stream_ref".
 */
guard_streams: string[]
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "seq".
 */
policy_seq: number
/**
 * Уровень подписи 0–3 (AD-13).
 */
signature_level: number
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "object_id".
 */
workplace_id?: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
workplace_session_id?: string
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "seq".
 */
seen_checkpoint?: number
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "timestamp".
 */
client_signed_at?: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "person_ref".
 */
on_behalf_of?: string
/**
 * Подписанный пакет команды — поле `signature` запроса, как есть (конверт DSSE, contracts/crypto/dsse-envelope.schema.json): подписан запрос JCS{operation, params, body} (Д-59). Принят модулем signing (CheckCommand) до записи; сам конверт записи — сервера. Его пересчитывает верификатор (AD-9). Отсутствует — демо без агента токена (Д-30).
 */
signature?: {

}
/**
 * Способ подписи команды: агент токена или ключ в браузере, бумага с заверением, демо-подписант сценариев.
 */
signature_method?: ("token_agent" | "paper" | "demo_signer")
/**
 * Класс хранения ключа подписанта по акту регистрации (AD-11, AD-14, Д-72): физический ключ или ключ в браузере. Берётся из реестра ключей, а не из заявления клиента.
 */
key_storage?: ("hardware_token" | "software_browser")
/**
 * Разновидность хранения ключа в браузере: в расширении или в хранилище страницы.
 */
key_storage_variant?: ("extension" | "page")
}
/**
 * Метаданные целостности под подписью (AD-10, кейс §6.3): версия формата, криптопрофиль, подписанты. Сами подписи — в конверте DSSE вокруг события.
 */
export interface Integrity {
/**
 * Версия формата подписанного пакета.
 */
format_version: number
/**
 * Криптопрофиль (AD-10): `gost`, `pq` (демонстрационный), `hybrid` (обе подписи обязательны).
 */
crypto_profile: ("gost" | "pq" | "hybrid")
/**
 * Обязательные подписанты пакета: `key_id@версия` каждого ключа; лишние подписи и повторы одного ключа не засчитываются.
 * 
 * @minItems 1
 * 
 * Items: Ссылка на ключ подписанта: `key_id@версия` (AD-10).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "key_ref".
 */
signers: [string, ...(string)[]]
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "digest".
 */
subject_ref?: string
/**
 * Открытые параметры проверки, если профиль их требует.
 */
verification_params?: {
[k: string]: string | undefined
}
}
/**
 * Сообщение SSE-канала живых обновлений (AD-21): сущность, её id и `seq` записи журнала, после которой она изменилась. Данных сущности в сообщении нет — фронтенд инвалидирует ключ Vue Query `[сущность, id]` и перечитывает её через API; после переподключения догоняет по `seq` (Last-Event-ID).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SseEntityChangedV1".
 */
export interface SseEntityChangedV1 {
/**
 * Вид сущности — первый элемент ключа Vue Query.
 */
entity: ("item" | "lot" | "nonconformity" | "incident" | "document" | "task" | "notification" | "workplace" | "equipment" | "process_version" | "analyzer_passport" | "erp_message" | "run" | "integrity" | "live_map" | "policy" | "quarantine" | "concession" | "process_hold" | "person" | "reference" | "key" | "material" | "partner")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
id: string
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 */
seq: number
/**
 * Идентификатор прогона сценария (AD-38); в профиле prod отсутствует.
 */
run_id?: string
/**
 * Режим ведущих портов, отдавших изменение (AD-36).
 */
mode?: ("fixtures" | "live")
}
/**
 * Изделие попало в «точку чистоты» (FR-49): после снятия стопа точки процесса первые N изделий, прошедших её, проходят усиленный контроль; межизделийная стадия адресует запись в поток изделия.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionCleanPointAssignedV1".
 */
export interface DecisionCleanPointAssignedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
hold_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id?: string
/**
 * Порядковый номер изделия после снятия (1…N).
 */
ordinal: number
/**
 * Сколько изделий под усиленным контролем (N).
 */
of: number
}
/**
 * Разрешение на отклонение выдано (ГОСТ Р ИСО 9000 п. 3.12.5, FR-54): номер, пункт КД/ТУ, область действия, лимит количества, срок; лимит открывается атомарно с записью, расход — атомарно с решением (AD-39).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionConcessionGrantedV1".
 */
export interface DecisionConcessionGrantedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
concession_id: string
/**
 * Номер разрешения по стандарту предприятия.
 */
number?: string
/**
 * Краткое содержание отклонения.
 */
title?: string
/**
 * Для какого решения по несоответствию.
 */
kind: ("repair" | "use_as_is")
/**
 * Пункт КД/ТУ, от которого разрешено отклонение.
 */
requirement_ref?: string
/**
 * Область действия — перечень изделий.
 * 
 * Items: Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "item_id".
 */
scope_item_ids?: string[]
/**
 * Область действия — диапазон номеров: от (включительно).
 */
scope_range_from?: string
/**
 * Область действия — диапазон номеров: до (включительно).
 */
scope_range_to?: string
/**
 * Лимит количества изделий.
 */
limit: number
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_until?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id?: string
/**
 * Подписи маршрута разрешения на момент записи; demo_stub — демо-профиль, подписи не проверялись.
 */
approvals_status?: ("route_closed" | "pending" | "demo_stub")
reason: Reason6
}
/**
 * Причина действия: код и текст.
 */
export interface Reason6 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Разрешение на отклонение отозвано — отзыв — новая запись; решения, принятые по разрешению, подсвечиваются (FR-54).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionConcessionRevokedV1".
 */
export interface DecisionConcessionRevokedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
concession_id: string
reason: Reason7
}
/**
 * Причина действия: код и текст.
 */
export interface Reason7 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Сдерживание применено правилом — утверждённое правило придержало изделие: наблюдать / доп. проверка / блок; автоматика изолирует, но не списывает (режим 2, FR-49, AD-27).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionContainmentAppliedV1".
 */
export interface DecisionContainmentAppliedV1 {
/**
 * Уровень сдерживания.
 */
level: ("none" | "observe" | "additional_check" | "item_hold" | "lot_hold")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
lot_id?: string
/**
 * Основания — `event_id` записей, на которые опирается запись.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
basis: string[]
}
/**
 * Сдерживание снято — разрешающее действие: только человеком или явно делегированным правилом; снятие блока ≠ годность (AD-27, AD-30).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionContainmentReleasedV1".
 */
export interface DecisionContainmentReleasedV1 {
/**
 * Записи сдерживания, которые снимаются.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
released_event_ids: string[]
reason: Reason8
}
/**
 * Причина действия: код и текст.
 */
export interface Reason8 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Сдерживание установлено человеком — уполномоченный ставит уровень сдерживания изделия (FR-49).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionContainmentSetV1".
 */
export interface DecisionContainmentSetV1 {
/**
 * Уровень сдерживания.
 */
level: ("none" | "observe" | "additional_check" | "item_hold" | "lot_hold")
reason: Reason9
}
/**
 * Причина действия: код и текст.
 */
export interface Reason9 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Решение по изделию применено правилом — типовая переделка по утверждённому техпроцессу по явно делегированному правилу режима 2 (PRD §11.9); необратимые решения правилом не принимаются.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionDispositionAppliedV1".
 */
export interface DecisionDispositionAppliedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
nc_id: string
/**
 * Только переделка.
 */
disposition: "rework"
}
/**
 * Решение по изделию принято — переделка / ремонт / как есть / списать / вернуть поставщику; ремонт и «как есть» — только по действующему разрешению на отклонение; исполнение — после закрытия маршрута подписей (FR-53, AD-43).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionDispositionSetV1".
 */
export interface DecisionDispositionSetV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
nc_id: string
/**
 * Вариант решения.
 */
disposition: ("rework" | "repair" | "use_as_is" | "scrap" | "return_to_supplier")
/**
 * Для списания: списание / переработка (разборка на годные части).
 */
scrap_kind?: ("writeoff" | "reprocess")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
concession_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id?: string
/**
 * Основание претензии для возврата поставщику.
 */
claim_basis?: string
reason: Reason10
/**
 * Подписи маршрута решения на момент записи (FR-50 режим 4, AD-43): route_closed — маршрут закрыт; pending — исполнение ждёт document.route.closed; demo_stub — демо-профиль, подписи не проверялись (заглушка порта «маршрут закрыт»).
 */
approvals_status?: ("route_closed" | "pending" | "demo_stub")
}
/**
 * Причина действия: код и текст.
 */
export interface Reason10 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Исполнение решения проверено — повторный контроль после переделки или ремонта пройден по всем методам операции: «исполнено» ≠ «проверено» (FR-53).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionDispositionVerifiedV1".
 */
export interface DecisionDispositionVerifiedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
nc_id: string
/**
 * Результаты повторного контроля.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
recheck_event_ids: string[]
}
/**
 * Изделие изолировано — контролёр изолировал изделие до решения; срок — по политике (FR-55). Положение «в изоляции» вычисляет process; физическое перемещение — `operation.movement.received` в изолятор.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionItemIsolatedV1".
 */
export interface DecisionItemIsolatedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
isolator_location_id?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
decision_due_at?: string
reason: Reason11
}
/**
 * Причина действия: код и текст.
 */
export interface Reason11 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Решение по партии на входном контроле — ЗТ-1: партия годна / годна частично / не годна / мало данных; без подписи партия в работу не идёт.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionLotResolvedV1".
 */
export interface DecisionLotResolvedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
lot_id: string
/**
 * Решение по партии.
 */
resolution: ("accept" | "accept_partially" | "reject" | "insufficient_data")
/**
 * Принятое количество при частичной приёмке.
 */
accepted_quantity?: number
/**
 * Результаты контроля партии.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
method_event_ids: string[]
reason?: Reason12
}
/**
 * Причина действия: код и текст.
 */
export interface Reason12 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Несоответствие закрыто по изделию — закрыто по изделию ≠ проблема устранена: расследование причины идёт отдельно (FR-51).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionNonconformityClosedV1".
 */
export interface DecisionNonconformityClosedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
nc_id: string
/**
 * Итог по изделию.
 */
summary?: string
}
/**
 * Несоответствие подтверждено — контролёр подтвердил: сигнал — несоответствие требованию КД (второй статус кейса §2.3). Исходный сигнал не меняется (FR-51, FR-52).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionNonconformityConfirmedV1".
 */
export interface DecisionNonconformityConfirmedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
nc_id: string
/**
 * Подтверждаемые сигналы.
 * 
 * @minItems 1
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
signal_ids: [string, ...(string)[]]
/**
 * Требование КД и факт.
 */
requirement_ref?: string
/**
 * Класс тяжести дефекта по ГОСТ 15467: критический / значительный / малозначительный / неизвестен.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "severity".
 */
severity: ("critical" | "major" | "minor" | "unknown")
/**
 * Вид дефекта по классификатору.
 */
defect_type_code?: string
/**
 * Нужен ли полный разбор причин (по политике).
 */
full_analysis_required?: boolean
reason: Reason13
}
/**
 * Причина действия: код и текст.
 */
export interface Reason13 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Черновик карточки несоответствия — система собрала карточку для контролёра: сигналы, обстоятельства до и после, чего не хватает (режим 1, FR-50). Те же данные — полезная нагрузка функции-намерения nonconformity «черновик несоответствия» (draft_nonconformity), которую вызывают quality и machinelogs.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionNonconformityDraftedV1".
 */
export interface DecisionNonconformityDraftedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
nc_id: string
/**
 * Сигналы карточки.
 * 
 * @minItems 1
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
signal_ids: [string, ...(string)[]]
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id?: string
/**
 * На чём основан сигнал.
 */
basis_kind?: ("inspection_result" | "equipment_deviation" | "check_skipped" | "damage_on_receipt" | "leak" | "special_process_violation" | "operator_report")
/**
 * Класс тяжести дефекта по ГОСТ 15467: критический / значительный / малозначительный / неизвестен.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "severity".
 */
severity?: ("critical" | "major" | "minor" | "unknown")
/**
 * Вид дефекта по классификатору; неизвестный — с флагом.
 */
defect_type_code?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
zone_id?: string
/**
 * Ключ шага процесса (`ant:properties/@stepKey`), к которому относится запись.
 */
step_key?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id?: string
/**
 * Требование КД: характеристика, допуск, ревизия.
 */
requirement_ref?: string
/**
 * Реакция карты реакций (FR-48): isolate — сдерживание правилом.
 */
reaction_outcome?: ("pass_to_next" | "manual_review" | "isolate" | "question_to_technologist")
/**
 * Карта реакций и строка: `‹id›@‹версия›#‹строка›`.
 */
reaction_map_ref?: string
/**
 * Закрывающая точка предъявления (`ZT-…`), если черновик возник на ней.
 */
closing_point?: string
/**
 * Номер предъявления, если черновик возник на точке предъявления.
 */
presentation_no?: number
/**
 * Нехватка сведений для разбора (перечисление, как у incident.hypothesis.computed).
 */
missing_information?: ("tool_unknown" | "cycle_end_time_unknown" | "no_observation_after_operation" | "no_observation_before_operation" | "operator_unknown" | "equipment_log_missing" | "other")[]
}
/**
 * Несоответствие зарегистрировано правилом — межизделийная стадия регистрирует несоответствие всем изделиям окна нарушения специального процесса, даже без найденного дефекта; решение — маршрутом комиссии (FR-151, AD-29).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionNonconformityRegisteredV1".
 */
export interface DecisionNonconformityRegisteredV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
nc_id: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
violation_window_event_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id: string
/**
 * Ключ шага процесса (`ant:properties/@stepKey`), к которому относится запись.
 */
step_key?: string
}
/**
 * Решение на точке предъявления — контролёр ОТК (или ПЗ/ВП) на закрывающей точке: принять / принять по разрешению на отклонение / не принять / мало данных. Без подписи изделие не проходит (FR-19, FR-44, FR-56).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionPresentationResolvedV1".
 */
export interface DecisionPresentationResolvedV1 {
/**
 * Ключ шага процесса (`ant:properties/@stepKey`), к которому относится запись.
 */
step_key: string
/**
 * Закрывающая точка: `ZT-1`…`ZT-6`, `ZT-R`, `ZT-V`.
 */
closing_point: string
/**
 * Решение; значение попадает в условие ветки BPMN `decision`.
 */
resolution: ("accept" | "accept_with_concession" | "reject" | "insufficient_data")
/**
 * Номер предъявления.
 */
presentation_no: number
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
concession_id?: string
/**
 * Результаты методов, на которых основано решение.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
method_event_ids: string[]
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id?: string
reason?: Reason14
}
/**
 * Причина действия: код и текст.
 */
export interface Reason14 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Пересмотр решения на закрывающей точке, принятого до новых данных (FR-32, FR-146, AD-5, Д-81): новая запись поверх прежней, прежняя не меняется. «Оставить в силе» — приёмка подтверждена на текущем состоянии; «отозвать приёмку» — основание приёмки не держится: блок изделия человеком, качество «не проверено», исправление результата контроля в учётной системе по решению человека (AD-7).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionPresentationReviewedV1".
 */
export interface DecisionPresentationReviewedV1 {
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
reviewed_event_id: string
/**
 * Шаг точки предъявления пересматриваемого решения.
 */
step_key: string
/**
 * Закрывающая точка пересматриваемого решения: `ZT-1`…`ZT-6`, `ZT-R`, `ZT-V`.
 */
closing_point: string
/**
 * Номер предъявления пересматриваемого решения.
 */
presentation_no: number
/**
 * Исход пересмотра: оставить решение в силе / отозвать приёмку.
 */
outcome: ("upheld" | "revoked")
/**
 * Новые факты, которые рассмотрены при пересмотре (причины метки «принято до новых данных»).
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
new_fact_ids: string[]
reason: Reason15
}
/**
 * Причина действия: код и текст.
 */
export interface Reason15 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Остановка точки процесса снята — снимает только уполномоченный; после снятия — «точка чистоты»: первые N изделий под усиленным контролем (FR-49).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionProcessHoldReleasedV1".
 */
export interface DecisionProcessHoldReleasedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
hold_id: string
/**
 * Сколько первых изделий под усиленным контролем.
 */
clean_point_items: number
reason: Reason16
}
/**
 * Причина действия: код и текст.
 */
export interface Reason16 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Остановка точки процесса установлена — сдерживание процесса (станок, инструмент, программа, операция) — отдельный объект от сдерживания изделий; ставит уполномоченный человек (FR-49).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionProcessHoldSetV1".
 */
export interface DecisionProcessHoldSetV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
hold_id: string
/**
 * Уровень.
 */
level: ("process_point_stop" | "critical_stop")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
tool_id?: string
/**
 * Программа.
 */
program_ref?: string
/**
 * Ключ шага процесса (`ant:properties/@stepKey`), к которому относится запись.
 */
step_key?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id?: string
/**
 * Условие снятия.
 */
release_condition?: string
reason: Reason17
}
/**
 * Причина действия: код и текст.
 */
export interface Reason17 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Назначена дополнительная проверка — данных мало для решения: метод, зона, срок; изделие ждёт в изоляции, автоматически не маршрутизируется (FR-52).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionRecheckRequestedV1".
 */
export interface DecisionRecheckRequestedV1 {
/**
 * Метод контроля (FR-14, FR-27): камера, КИМ, рентген, УЗК, капиллярный, течеискатель, момент затяжки, визуальный человеком, документы поставщика, лаборатория, иное.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "inspection_method".
 */
method: ("camera" | "cmm" | "radiography" | "ultrasonic" | "penetrant" | "leak_test" | "torque" | "visual_human" | "supplier_documents" | "laboratory" | "other")
/**
 * Зоны доп. проверки.
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
zone_ids?: string[]
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
due_at?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
suggested_by_rule?: string
reason: Reason18
}
/**
 * Причина действия: код и текст.
 */
export interface Reason18 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Разрешено превысить лимит доработок — сверх лимита доработок зоны — только по отдельному разрешению уполномоченного (FR-18); без него гард отвечает `process.rework_limit_exceeded`.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionReworkLimitWaivedV1".
 */
export interface DecisionReworkLimitWaivedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
zone_id: string
/**
 * Сколько доработок уже было.
 */
used: number
/**
 * Лимит по нормативному слою.
 */
limit: number
/**
 * Сколько дополнительных доработок разрешено.
 */
extra_allowed: number
reason: Reason19
}
/**
 * Причина действия: код и текст.
 */
export interface Reason19 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Сигнал отклонён — ложное срабатывание; причина обязательна; исходный сигнал остаётся в истории (FR-52, кейс §5.1 «Решение пользователя»).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DecisionSignalRejectedV1".
 */
export interface DecisionSignalRejectedV1 {
/**
 * Отклоняемые сигналы.
 * 
 * @minItems 1
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
signal_ids: [string, ...(string)[]]
reason: Reason20
/**
 * Снимок — кандидат в размеченные данные для настройки модели (FR-99).
 */
label_for_adaptation?: boolean
}
/**
 * Причина действия: код и текст.
 */
export interface Reason20 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Статус бумажного экземпляра — напечатан / подписан / уничтожен; журнал не трогается (AD-12).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DocumentPaperStatusChangedV1".
 */
export interface DocumentPaperStatusChangedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id: string
/**
 * Версия.
 */
version: number
/**
 * Статус бумажного экземпляра.
 */
paper_status: ("printed" | "signed" | "destroyed")
/**
 * Номер экземпляра.
 */
copy_no?: string
}
/**
 * Маршрут подписей закрыт — все обязательные подписи проверены над текущим отпечатком: полномочие и клеймо на `seq`, разделение обязанностей; модули-исполнители действуют только по этой реакции (AD-43).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DocumentRouteClosedV1".
 */
export interface DocumentRouteClosedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id: string
/**
 * Версия.
 */
version: number
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
doc_digest: string
/**
 * Засчитанные подписи.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
signature_event_ids: string[]
/**
 * Как проверены подписи: full — полномочие и клеймо по политике на seq, криптопроверка подписи (эпик 27); demo — демо-профиль: подписи без агента токена, полномочие по стартовой политике, криптопроверка не проводилась (Д-30).
 */
verification?: ("full" | "demo")
}
/**
 * Подписант не согласовал версию документа — вернул с замечанием; маршрут этой версии не закрывается, нужна новая версия или аннулирование (FR-136, AD-43).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DocumentSignatureDeclinedV1".
 */
export interface DocumentSignatureDeclinedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id: string
/**
 * Версия документа.
 */
version: number
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
doc_digest: string
/**
 * Этап маршрута.
 */
stage: number
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
signer_person_id: string
/**
 * Текст на русском для человека.
 */
comment: string
}
/**
 * Подпись документа записана — подпись над отпечатком: агентом токена или на бумаге с заверением (скан, QR, заверитель ≠ подписант, учётный номер оригинала) (FR-139, AD-43).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DocumentSignatureRecordedV1".
 */
export interface DocumentSignatureRecordedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id: string
/**
 * Версия документа.
 */
version: number
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
doc_digest: string
/**
 * Этап маршрута.
 */
stage: number
/**
 * Способ подписи.
 */
method: ("token_agent" | "paper" | "device" | "demo_signer")
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
signer_person_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
authority_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
stamp_id?: string
/**
 * Уровень подписи.
 */
signature_level: number
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
attested_by?: string
/**
 * Учётный номер бумажного оригинала в архиве ОТК.
 */
paper_original_no?: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
scan_address?: string
/**
 * Сводка.
 * 
 * @maxItems 7
 */
summary?: []|[SummaryField]|[SummaryField, SummaryField]|[SummaryField, SummaryField, SummaryField]|[SummaryField, SummaryField, SummaryField, SummaryField]|[SummaryField, SummaryField, SummaryField, SummaryField, SummaryField]|[SummaryField, SummaryField, SummaryField, SummaryField, SummaryField, SummaryField]|[SummaryField, SummaryField, SummaryField, SummaryField, SummaryField, SummaryField, SummaryField]
/**
 * Ключ подписанта `key_id@версия` (агент токена); проверяет модуль signing (эпик 27).
 */
key_ref?: string
/**
 * Подпись агента над отпечатком документа (base64, DSSE PAE класса document-signature); проверяет модуль signing (эпик 27). Пусто — демо без агента токена (Д-30).
 */
signature?: string
}
/**
 * Поле сводки уровня 2 (3–7 полей) — как видел подписант.
 */
export interface SummaryField {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}
/**
 * Версия документа аннулирована — аннулирование — новой записью; подписи прежней версии остаются при ней (AD-12).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DocumentVersionAnnulledV1".
 */
export interface DocumentVersionAnnulledV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id: string
/**
 * Версия.
 */
version: number
reason: Reason21
}
/**
 * Причина действия: код и текст.
 */
export interface Reason21 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Версия документа сформирована — документ = шаблон шага@версия × события-источники; отпечаток и набор обязательных подписей (замороженный, AD-43) входят в канонический JSON (AD-12).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DocumentVersionDraftedV1".
 */
export interface DocumentVersionDraftedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id: string
/**
 * Версия документа.
 */
version: number
/**
 * Шаблон `‹id›@‹версия›`.
 */
template_ref: string
/**
 * Версия формата документа.
 */
doc_format_version: number
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
doc_digest: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
rendering_hash: string
/**
 * Поток записи (AD-39): `item:‹item_id›`, `‹вид_объекта›:‹id›` или `global`.
 */
subject_ref: string
/**
 * События-источники.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
source_event_ids: string[]
/**
 * Замороженный набор обязательных подписей.
 * 
 * @minItems 1
 */
required_approvals: [ApprovalStage, ...(ApprovalStage)[]]
/**
 * Заменяемая версия документа.
 */
supersedes_document_version?: number
/**
 * Название документа для людей (из шаблона).
 */
title?: string
}
/**
 * Этап маршрута подписей (AD-13, AD-43).
 */
export interface ApprovalStage {
/**
 * Номер этапа.
 */
stage: number
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
authority_id: string
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 */
stamp_kind?: string
/**
 * Сколько подписей.
 */
quorum: ("one" | "all" | "k_of_n")
/**
 * k для k из n.
 */
k?: number
/**
 * Уровень подписи.
 */
signature_level: number
/**
 * Разрешена бумага с заверением.
 */
paper_allowed: boolean
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
attester_authority_id?: string
/**
 * Внешняя сторона.
 */
external_party?: ("none" | "customer_representative" | "partner")
/**
 * Кто подписывает этап — для людей (из маршрута шаблона).
 */
title?: string
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 */
role?: string
/**
 * Сколько засчитанных подписей нужно на этапе.
 */
required_count?: number
/**
 * Этап закрывает само решение-источник документа (его автор подписал решение уровнем этапа) — отдельной подписи не требуется.
 */
by_source?: boolean
/**
 * Правила разделения обязанностей этапа: distinct_signers — один человек подписывает один этап; not_item_participant — не участвовал в изготовлении изделия (FR-56).
 * 
 * Items: Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
separation?: string[]
}
/**
 * Запрошено оформление документа — человек запрашивает документ с маршрутом: «Запросить решение», выдача прав, назначение контролёра (FR-136, AD-12).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "DocumentVersionRequestedV1".
 */
export interface DocumentVersionRequestedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id: string
/**
 * Шаблон `‹id›@‹версия›` из нормативного слоя.
 */
template_ref: string
/**
 * Поток записи (AD-39): `item:‹item_id›`, `‹вид_объекта›:‹id›` или `global`.
 */
subject_ref: string
/**
 * События-источники.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
source_event_ids?: string[]
/**
 * Решение, которое оформляется документом «Запросить решение» (например, `disposition=scrap`; FR-146).
 */
decision?: string
/**
 * Комментарий автора запроса.
 */
comment?: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
requested_by?: string
}
/**
 * Сводка параметров за цикл — edge-агент на окно цикла отдаёт сводку (среднее, максимум, выход за уставку) — не миллисекундную телеметрию; сырые данные остаются на краю (FR-147, AD-25). `event_id` = UUIDv5(устройство, окно, вид сводки).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "EquipmentCycleSummarizedV1".
 */
export interface EquipmentCycleSummarizedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
station_id?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
window_start: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
window_end: string
/**
 * Идентификатор цикла у источника.
 */
cycle_ref?: string
/**
 * Сводки параметров.
 * 
 * @minItems 1
 */
parameters: [ParameterSummary, ...(ParameterSummary)[]]
/**
 * Ссылка на сырые данные на краю или в историческом хранилище.
 */
raw_ref?: string
}
/**
 * Сводка одного параметра.
 */
export interface ParameterSummary {
/**
 * Параметр (ток, напряжение, скорость, давление, момент…).
 */
parameter: string
mean?: Measurement1
max?: Measurement2
min?: Measurement3
setpoint?: Tolerance1
/**
 * Сколько миллисекунд вне уставки.
 */
out_of_setpoint_ms?: number
}
/**
 * Измерение без float: значение = value × 10^(−scale) в единице unit.
 */
export interface Measurement1 {
/**
 * Целая мантисса значения.
 */
value: number
/**
 * Число знаков после запятой (масштаб).
 */
scale: number
/**
 * Единица измерения, код UCUM (например, `A`, `V`, `mm`, `N.m`, `Pa`, `min`).
 */
unit: string
}
/**
 * Измерение без float: значение = value × 10^(−scale) в единице unit.
 */
export interface Measurement2 {
/**
 * Целая мантисса значения.
 */
value: number
/**
 * Число знаков после запятой (масштаб).
 */
scale: number
/**
 * Единица измерения, код UCUM (например, `A`, `V`, `mm`, `N.m`, `Pa`, `min`).
 */
unit: string
}
/**
 * Измерение без float: значение = value × 10^(−scale) в единице unit.
 */
export interface Measurement3 {
/**
 * Целая мантисса значения.
 */
value: number
/**
 * Число знаков после запятой (масштаб).
 */
scale: number
/**
 * Единица измерения, код UCUM (например, `A`, `V`, `mm`, `N.m`, `Pa`, `min`).
 */
unit: string
}
/**
 * Допуск: номинал и границы; отсутствующая граница — не ограничена.
 */
export interface Tolerance1 {
nominal?: Measurement
lower?: Measurement
upper?: Measurement
}
/**
 * Отклонение оборудования выделено — выделитель значимых событий: параметр вне уставки, перегрузка, предупреждение ресурса инструмента, ручное изменение режима (подача 130 %), внеплановая смена программы; самостоятельный сигнал даже без дефекта (FR-121, FR-147).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "EquipmentDeviationDetectedV1".
 */
export interface EquipmentDeviationDetectedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
station_id?: string
/**
 * Вид отклонения.
 */
deviation_kind: ("out_of_setpoint" | "overload" | "tool_life_warning" | "manual_override" | "unplanned_program_change" | "alarm" | "other")
/**
 * Параметр.
 */
parameter?: string
value?: Measurement4
setpoint?: Tolerance2
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
started_at: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
ended_at?: string
/**
 * Код у источника.
 */
code?: string
}
/**
 * Измерение без float: значение = value × 10^(−scale) в единице unit.
 */
export interface Measurement4 {
/**
 * Целая мантисса значения.
 */
value: number
/**
 * Число знаков после запятой (масштаб).
 */
scale: number
/**
 * Единица измерения, код UCUM (например, `A`, `V`, `mm`, `N.m`, `Pa`, `min`).
 */
unit: string
}
/**
 * Допуск: номинал и границы; отсутствующая граница — не ограничена.
 */
export interface Tolerance2 {
nominal?: Measurement
lower?: Measurement
upper?: Measurement
}
/**
 * Событие оборудования привязано к выполнению операции — межизделийная стадия по оборудованию и интервалу выполнения отнесла событие оборудования (без изделия) к выполнению операции изделия и адресовала его в поток изделия; из таких записей свёртка изделия строит профиль выполнения операции (FR-121, FR-148, AD-29, AD-42). Привязку делает только стадия; edge-агент `operation_run_id` не подставляет. Данные исходного события копируются как есть (`subject_data`), исходная запись остаётся в потоке оборудования.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "EquipmentEventBoundV1".
 */
export interface EquipmentEventBoundV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id: string
/**
 * Как привязано: событие в интервале выполнения / обстановка до начала (последние программа, инструмент, состояние).
 */
binding: ("interval" | "context")
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
subject_event_id: string
/**
 * Тип исходного события (семейство equipment).
 */
subject_event_type: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
subject_occurred_at: string
/**
 * Источник исходного события (FR-140): ручной ввод, станок, датчик, внешняя система.
 */
subject_source_kind?: string
/**
 * Данные исходного события (текущая версия схемы его типа).
 */
subject_data: {

}
}
/**
 * Программа оборудования сменилась — программа и её ревизия (MTConnect EVENT); смена программы — повод для акта первой детали (FR-147).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "EquipmentProgramChangedV1".
 */
export interface EquipmentProgramChangedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
station_id?: string
/**
 * Программа.
 */
program_ref: string
/**
 * Ревизия программы.
 */
program_revision: string
/**
 * Плановая смена (иначе — отклонение).
 */
planned: boolean
}
/**
 * Состояние оборудования изменилось (v1) — первая версия: одно поле `machine_state`, смешивающее режим работы и исправность (как в каталоге процессной сессии и кейсе §4.4). Сохраняется для чтения старых записей; повышатель v1 → v2 разделяет поле по MTConnect.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "EquipmentStateChangedV1".
 */
export interface EquipmentStateChangedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
station_id?: string
/**
 * Состояние: работает / ожидает / остановлено / предупреждение / авария.
 */
machine_state: ("running" | "idle" | "stopped" | "warning" | "alarm")
/**
 * Код состояния или аварии у источника.
 */
code?: string
/**
 * Параметр, если предупреждение по параметру.
 */
parameter?: string
value?: Measurement5
setpoint?: Tolerance3
}
/**
 * Измерение без float: значение = value × 10^(−scale) в единице unit.
 */
export interface Measurement5 {
/**
 * Целая мантисса значения.
 */
value: number
/**
 * Число знаков после запятой (масштаб).
 */
scale: number
/**
 * Единица измерения, код UCUM (например, `A`, `V`, `mm`, `N.m`, `Pa`, `min`).
 */
unit: string
}
/**
 * Допуск: номинал и границы; отсутствующая граница — не ограничена.
 */
export interface Tolerance3 {
nominal?: Measurement
lower?: Measurement
upper?: Measurement
}
/**
 * Состояние оборудования изменилось — один тип на смысл от любого источника (ручной ввод, датчик, контроллер, ЧПУ, OPC UA, MTConnect) с пометкой источника; классификация MTConnect: EXECUTION, CONTROLLER_MODE, CONDITION (FR-147, AD-29). v2: несовместимо с v1 — `machine_state` разделён.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "EquipmentStateChangedV2".
 */
export interface EquipmentStateChangedV2 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
station_id?: string
/**
 * Режим работы (MTConnect EXECUTION).
 */
execution: ("running" | "idle" | "stopped" | "setup" | "interrupted" | "unknown")
/**
 * Режим управления (авто / ручной).
 */
controller_mode: ("automatic" | "manual" | "manual_data_input" | "unknown")
/**
 * Исправность (MTConnect CONDITION): норма / предупреждение / неисправность.
 */
condition: ("normal" | "warning" | "fault" | "unknown")
/**
 * Код состояния или аварии у источника.
 */
code?: string
/**
 * Параметр, по которому предупреждение или неисправность.
 */
parameter?: string
value?: Measurement6
setpoint?: Tolerance4
}
/**
 * Измерение без float: значение = value × 10^(−scale) в единице unit.
 */
export interface Measurement6 {
/**
 * Целая мантисса значения.
 */
value: number
/**
 * Число знаков после запятой (масштаб).
 */
scale: number
/**
 * Единица измерения, код UCUM (например, `A`, `V`, `mm`, `N.m`, `Pa`, `min`).
 */
unit: string
}
/**
 * Допуск: номинал и границы; отсутствующая граница — не ограничена.
 */
export interface Tolerance4 {
nominal?: Measurement
lower?: Measurement
upper?: Measurement
}
/**
 * Инструмент установлен или заменён — инструмент и его ресурс (например, 73/75); замена инструмента сужает окно области риска (FR-147, FR-61).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "EquipmentToolChangedV1".
 */
export interface EquipmentToolChangedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
station_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
tool_id: string
/**
 * Сделано циклов.
 */
tool_life_used?: number
/**
 * Допустимо циклов.
 */
tool_life_limit?: number
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
fixture_id?: string
}
/**
 * Окно нарушения режима определено — по журналам оборудования определено окно нарушения режима специального процесса; стадия регистрирует несоответствие всем изделиям окна (FR-151).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "EquipmentViolationWindowResolvedV1".
 */
export interface EquipmentViolationWindowResolvedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id: string
/**
 * Ключ шага процесса (`ant:properties/@stepKey`), к которому относится запись.
 */
step_key?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
window_start: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
window_end: string
/**
 * Отклонения, образующие окно.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
deviation_event_ids: string[]
/**
 * Выполнения операции в окне.
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
affected_operation_run_ids: string[]
/**
 * Изделия выполнений в окне — каждому стадия регистрирует несоответствие, даже без найденного дефекта (FR-151).
 * 
 * Items: Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
affected_item_ids?: string[]
}
/**
 * Партия поступила — на склад пришла партия: поставщик, номер партии и плавки, количество, сертификат; ещё не проверена — запускать нельзя (FR-91).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ErpLotReceivedV1".
 */
export interface ErpLotReceivedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
lot_id: string
/**
 * Система.
 */
external_system: ("onec" | "galaktika")
/**
 * Номер приходного документа.
 */
external_number: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
supplier_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
item_type_id: string
/**
 * Плавка.
 */
heat_no?: string
/**
 * Количество.
 */
quantity: number
/**
 * Номер сертификата; отсутствует — сертификата нет.
 */
certificate_no?: string
/**
 * Срок годности, если есть.
 */
expiry_date?: string
}
/**
 * Справочник номенклатуры получен — номенклатура и состав изделия из учётной системы; расхождение обозначений — отчёт, без автосоздания (FR-91).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ErpNomenclatureSyncedV1".
 */
export interface ErpNomenclatureSyncedV1 {
/**
 * Система.
 */
external_system: ("onec" | "galaktika")
/**
 * Позиции.
 * 
 * @minItems 1
 */
entries: [NomenclatureEntry, ...(NomenclatureEntry)[]]
}
/**
 * Позиция номенклатуры.
 */
export interface NomenclatureEntry {
/**
 * ID во внешней системе.
 */
external_id: string
/**
 * Обозначение.
 */
designation: string
/**
 * Наименование.
 */
name: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
item_type_id?: string
}
/**
 * Получено производственное задание — из 1С (или Галактики) пришло задание: номенклатура, количество, срок, ревизии КД и ТП; повтор с тем же номером второй раз не учитывается (FR-91).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ErpOrderReceivedV1".
 */
export interface ErpOrderReceivedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
order_id: string
/**
 * Система.
 */
external_system: ("onec" | "galaktika")
/**
 * Номер задания во внешней системе.
 */
external_number: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
item_type_id: string
/**
 * Количество.
 */
quantity: number
/**
 * Срок.
 */
due_date?: string
/**
 * Ревизия КД и ТП.
 */
item_revision?: string
}
/**
 * Решение об исправлении отправленного — новая версия реакции с тем же бизнес-ключом и другим содержимым уходит исправлением (сторно + новое) только по решению человека (AD-7).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ErpPostingCompensationDecidedV1".
 */
export interface ErpPostingCompensationDecidedV1 {
/**
 * Бизнес-ключ.
 */
business_key: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
superseded_request_event_id: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
new_request_event_id?: string
/**
 * Решение.
 */
decision: ("send_correction" | "keep_as_sent")
reason: Reason22
}
/**
 * Причина действия: код и текст.
 */
export interface Reason22 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Исходящее сообщение в карантине — повтор при транспортных ошибках исчерпан или ошибка данных без автоповтора — задача администратору (FR-96).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ErpPostingQuarantinedV1".
 */
export interface ErpPostingQuarantinedV1 {
/**
 * Бизнес-ключ.
 */
business_key: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
request_event_id: string
/**
 * Попыток.
 */
attempts: number
/**
 * Код последней ошибки.
 */
last_error_code?: string
/**
 * Почему в карантине: повторы при транспортных ошибках исчерпаны / ошибка данных без автоповтора / несовместимый контракт / новая версия отправленного ждёт решения человека (AD-7).
 */
cause?: ("transport_exhausted" | "data_error" | "contract_incompatible" | "correction_pending")
/**
 * Текст последней ошибки.
 */
error_message?: string
/**
 * Версия содержимого, которая в карантине.
 */
message_version?: number
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
item_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
lot_id?: string
}
/**
 * Учётное сообщение сформировано — исходящее учётное действие на закрывающей точке с бизнес-ключом идемпотентности (субъект, действие, точка); очередь отправки — проекция журнала; при воспроизведении ничего не отправляется (AD-7, AD-18).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ErpPostingRequestedV1".
 */
export interface ErpPostingRequestedV1 {
/**
 * Бизнес-ключ: субъект, учётное действие, закрывающая точка.
 */
business_key: string
/**
 * Система.
 */
external_system: ("onec" | "galaktika")
/**
 * Учётное действие порта учёта; `return_from_defect` — «возврат из брака в производство» после удачной переделки или ремонта (решение Д-17).
 */
action: ("accept_into_work" | "warehouse_transfer" | "scrap_transfer_rework" | "scrap_transfer_writeoff" | "scrap_transfer_reprocess" | "return_to_supplier" | "release" | "inspection_result" | "return_from_defect")
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
item_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
lot_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
from_warehouse_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
to_warehouse_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
concession_id?: string
/**
 * Выпуск после переделки.
 */
after_rework?: boolean
/**
 * Основание претензии для возврата.
 */
claim_basis?: string
/**
 * Версия содержимого по бизнес-ключу; другое содержимое — только исправлением по решению человека.
 */
message_version: number
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
message_id?: string
/**
 * Шаг процесса, на котором сформировано действие (событие-сообщение BPMN или точка предъявления).
 */
step_key?: string
/**
 * Закрывающая точка — часть бизнес-ключа: `ZT-1`…`ZT-6`, `ZT-R`, шаг процесса или цикл брака `defect.‹N›`.
 */
closing_point?: string
/**
 * Записи-основания: событие-сообщение процесса, решение на закрывающей точке, решение по несоответствию.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
basis_event_ids?: string[]
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
order_id?: string
/**
 * Количество (для партии).
 */
quantity?: number
/**
 * Итог контроля для «результата контроля»: годно / годно по разрешению на отклонение / годно частично / не годно / мало данных.
 */
resolution?: ("accept" | "accept_with_concession" | "accept_partially" | "reject" | "insufficient_data")
/**
 * Номер предъявления.
 */
presentation_no?: number
/**
 * Несоответствия — основание перевода в брак, возврата или «не годно».
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "object_id".
 */
nc_ids?: string[]
}
/**
 * Переотправка запрошена администратором — ручная переотправка сообщения из карантина с тем же бизнес-ключом (FR-96).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ErpPostingResendRequestedV1".
 */
export interface ErpPostingResendRequestedV1 {
/**
 * Бизнес-ключ.
 */
business_key: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
request_event_id: string
reason: Reason23
}
/**
 * Причина действия: код и текст.
 */
export interface Reason23 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Ответ учётной системы — квитанция или ошибка 1С на исходящее сообщение; ось «учёт в 1С» меняется только по подтверждению (AD-30).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ErpPostingRespondedV1".
 */
export interface ErpPostingRespondedV1 {
/**
 * Бизнес-ключ.
 */
business_key: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
request_event_id: string
/**
 * Итог.
 */
outcome: ("accepted" | "duplicate" | "rejected")
/**
 * Документ 1С.
 */
external_document_ref?: string
/**
 * Код ошибки из contracts/errors.yaml (например, `reference.not_found`).
 */
error_code?: string
/**
 * Текст ошибки внешней системы.
 */
error_message?: string
/**
 * Статус учёта изделия после подтверждения.
 */
resulting_status?: ("not_sent" | "accepted_into_work" | "moved" | "transferred_to_scrap" | "returned_to_supplier" | "released")
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
message_id?: string
/**
 * Учётное действие сообщения.
 */
action?: ("accept_into_work" | "warehouse_transfer" | "scrap_transfer_rework" | "scrap_transfer_writeoff" | "scrap_transfer_reprocess" | "return_to_supplier" | "release" | "inspection_result" | "return_from_defect")
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
item_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
lot_id?: string
/**
 * Номер квитанции учётной системы; повтор с тем же номером сообщения возвращает ту же квитанцию.
 */
receipt?: string
/**
 * Код ответа HTTP учётной системы.
 */
http_status?: number
/**
 * Номер попытки отправки, на которую пришёл ответ.
 */
attempt?: number
}
/**
 * Выписка паспорта партнёра принята — получатель сам проверил подписи до корней партнёра (класс `partner`); непроверяемая — «происхождение не подтверждено»; изменённая — отклонена (FR-132, AD-19).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "FederationExtractReceivedV1".
 */
export interface FederationExtractReceivedV1 {
/**
 * Отправитель.
 */
partner_code: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
extract_digest: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
lot_id?: string
/**
 * Плавка.
 */
heat_no?: string
/**
 * Статус происхождения.
 */
origin_status: ("verified" | "server_confirmed_only" | "unverified")
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
material_address: string
}
/**
 * Партнёр подтвердил получение — квитанция событием — работает и через однонаправленные каналы (FR-131).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "FederationMessageAcknowledgedV1".
 */
export interface FederationMessageAcknowledgedV1 {
/**
 * Партнёр.
 */
partner_code: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
message_id: string
/**
 * Итог.
 */
outcome: ("received" | "rejected")
}
/**
 * Межзаводское сообщение отправлено — выписка или адресное уведомление партнёру (область риска задела отгруженное, претензия поставщику) (FR-133).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "FederationMessageSentV1".
 */
export interface FederationMessageSentV1 {
/**
 * Получатель.
 */
partner_code: string
/**
 * Вид.
 */
kind: ("passport_extract" | "risk_notice" | "claim_notice")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id?: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
message_id: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
extract_digest?: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
material_address?: string
/**
 * Изделие или партия выписки (локальный ID).
 */
subject_id?: string
}
/**
 * Партнёр зарегистрирован — код предприятия и корни доверия партнёра; администратор безопасности + начальник ОТК (AD-19).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "FederationPartnerRegisteredV1".
 */
export interface FederationPartnerRegisteredV1 {
/**
 * Код предприятия-партнёра.
 */
partner_code: string
/**
 * Название.
 */
name: string
/**
 * Отпечатки корневых ключей партнёра.
 * 
 * @minItems 1
 * 
 * Items: Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "digest".
 */
root_fingerprints: [string, ...(string)[]]
/**
 * Адрес порта межзаводского обмена.
 */
endpoint?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id: string
}
/**
 * Сдерживание распространено по генеалогии — адресованная запись стадии изделию: блок партии доходит до изделий из партии и собранных из них; блок компонента — вверх по дереву сборки (AD-42). Ось «сдерживание» меняет nonconformity по этой записи (AD-30); снятие основания блок не снимает — решает человек (AD-27).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "GenealogyContainmentPropagatedV1".
 */
export interface GenealogyContainmentPropagatedV1 {
/**
 * Уровень сдерживания источника.
 */
level: ("none" | "observe" | "additional_check" | "item_hold" | "lot_hold")
/**
 * Откуда пришло: блок партии, блок компонента (вверх по сборке), блок исходного изделия при разделении.
 */
source: ("lot" | "component" | "split_parent")
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
source_event_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
lot_id?: string
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
source_item_id?: string
/**
 * Путь по дереву сборки от источника к изделию.
 * 
 * Items: Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
path?: string[]
/**
 * Основание снято у источника: блок остаётся до решения человека (AD-27, AD-3).
 */
released?: boolean
/**
 * Основания — `event_id` записей.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 */
basis: string[]
}
/**
 * Временная группа расформирована — разгруппировка (FR-15).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "GenealogyGroupDissolvedV1".
 */
export interface GenealogyGroupDissolvedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
group_id: string
}
/**
 * Временная группа сформирована — садка или групповая операция над N изделиями: общий журнал режима, результат образца-свидетеля распространяется на группу (FR-15).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "GenealogyGroupFormedV1".
 */
export interface GenealogyGroupFormedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
group_id: string
/**
 * Вид группы.
 */
kind: ("charge" | "batch_operation" | "transport" | "other")
/**
 * Изделия группы.
 * 
 * @minItems 1
 * 
 * Items: Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
item_ids: [string, ...(string)[]]
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
witness_item_id?: string
}
/**
 * Связь генеалогии добавлена — адресованная запись обоим изделиям: компонент ← сборка, партия ← изделие; запросы вверх и вниз по дереву (FR-45, AD-42).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "GenealogyLinkAddedV1".
 */
export interface GenealogyLinkAddedV1 {
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
parent_item_id?: string
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
child_item_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
lot_id?: string
/**
 * Вид связи.
 */
relation: ("component_of" | "made_from_lot" | "split_from" | "grouped_with")
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
basis_event_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
group_id?: string
/**
 * Позиция компонента в сборке по спецификации.
 */
position?: string
/**
 * Связь перенесена при разделении 1→N или по сборке (происхождение компонента), а не записана напрямую (FR-15).
 */
inherited?: boolean
}
/**
 * Партия выдана в производство — какие изделия или задания получили материал или покупные из партии — связь «партия → изделия» для области риска (FR-45).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "GenealogyLotIssuedV1".
 */
export interface GenealogyLotIssuedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
lot_id: string
/**
 * Изделия-получатели.
 * 
 * Items: Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
item_ids?: string[]
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
order_id?: string
/**
 * Количество.
 */
quantity: number
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
issued_by: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
to_location_id?: string
}
/**
 * Партия принята на входной контроль — партия физически у ОТК: количество, упаковка, сертификат есть или нет; начинается срок входного контроля.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "GenealogyLotRegisteredV1".
 */
export interface GenealogyLotRegisteredV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
lot_id: string
/**
 * Фактическое количество.
 */
actual_quantity: number
/**
 * Упаковка без повреждений.
 */
packaging_ok?: boolean
/**
 * Сертификат поставщика есть.
 */
certificate_present: boolean
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
registered_by: string
/**
 * Вид: партия материала или покупных / плавка (FR-45). Садка — временная группа genealogy.group.formed (kind = charge).
 */
lot_kind?: ("lot" | "heat")
/**
 * Номер плавки партии (запрос «плавка → все изделия», FR-45).
 */
heat_no?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
item_type_id?: string
}
/**
 * Результат образца-свидетеля распространён на изделие группы — адресованная запись стадии каждому изделию садки или групповой операции: результат контроля свидетеля виден в паспортах всех изделий группы (FR-15, AD-42).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "GenealogyWitnessPropagatedV1".
 */
export interface GenealogyWitnessPropagatedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
group_id: string
/**
 * Вид группы.
 */
group_kind: ("charge" | "batch_operation" | "transport" | "other")
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
witness_item_id: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
inspection_event_id: string
/**
 * Исход контроля свидетеля.
 */
outcome: ("defect_indicated" | "no_defect_indicated" | "unable_to_assess")
/**
 * Метод контроля свидетеля.
 */
method?: string
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "step_key".
 */
step_key?: string
/**
 * Точка контроля.
 */
inspection_point?: string
/**
 * Номер заключения или протокола испытаний.
 */
conclusion_ref?: string
}
/**
 * Мера назначена — коррекция или корректирующее действие; без плана проверки результативности мера не создаётся (FR-64).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentActionAssignedV1".
 */
export interface IncidentActionAssignedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
action_id: string
/**
 * Тип меры.
 */
action_type: ("correction" | "corrective_action" | "preventive_action")
/**
 * Направление.
 */
direction: ("prevent_occurrence" | "improve_detection")
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
owner_id: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
due_at?: string
effectiveness_plan: EffectivenessPlan
/**
 * Что делается — словами («проверка инструмента каждые 75 циклов», «обучение сварщиков»); основа организационной памяти (FR-138).
 */
title?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
suggestion_id?: string
}
/**
 * План проверки результативности.
 */
export interface EffectivenessPlan {
/**
 * Метрика.
 */
metric: string
/**
 * Базовый уровень.
 */
baseline: string
/**
 * Окно наблюдения, дней.
 */
window_days: number
/**
 * Критерий успеха.
 */
success_criterion: string
/**
 * Временно усиленный контроль.
 */
enhanced_control?: string
}
/**
 * Результативность меры оценена — сравнение метрики до и после; провал — мера снова открыта, разбор заново (FR-64).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentActionEvaluatedV1".
 */
export interface IncidentActionEvaluatedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
action_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id: string
/**
 * Итог.
 */
result: ("effective" | "failed")
/**
 * Данные окна.
 */
evidence?: string
}
/**
 * Мера внедрена — «внедрено» ≠ «эффективно»: начинается окно наблюдения (FR-64).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentActionImplementedV1".
 */
export interface IncidentActionImplementedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
action_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id: string
/**
 * Что сделано.
 */
note?: string
}
/**
 * Решено, нужен ли полный разбор — по политике: тяжесть, повторяемость, критичность, размер области; единичный мелкий случай — коррекция и закрытие (FR-64).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentAnalysisScopedV1".
 */
export interface IncidentAnalysisScopedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id: string
/**
 * Нужен полный разбор.
 */
full_analysis: boolean
reason: Reason24
}
/**
 * Причина действия: код и текст.
 */
export interface Reason24 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Вывод о причине — причина подтверждена или «не установлена» с обоснованием; обязателен ответ «чем проверили» (FR-59). Необратимое инженерное решение (AD-27).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentCauseConcludedV1".
 */
export interface IncidentCauseConcludedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id: string
/**
 * Несоответствия.
 * 
 * @minItems 1
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
nc_ids: [string, ...(string)[]]
/**
 * Итог.
 */
conclusion: ("confirmed" | "not_established")
/**
 * Категория подтверждённой причины.
 */
category?: ("incoming" | "equipment" | "performer" | "handling" | "assembly" | "documentation" | "not_established")
/**
 * Чем проверили.
 */
verification: string
reason: Reason25
/**
 * Ветка причины: почему возник (why_made) или почему не обнаружили раньше (why_missed). Нет — why_made.
 */
branch?: ("why_made" | "why_missed")
}
/**
 * Причина действия: код и текст.
 */
export interface Reason25 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Гипотезы и обстоятельства пересчитаны — версия вывода разбора: окно возможного возникновения, кандидаты-обстоятельства, альтернативы, нехватка сведений; при позднем событии — новая версия (FR-58, FR-59, AD-3).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentHypothesisComputedV1".
 */
export interface IncidentHypothesisComputedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
nc_id: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
causal_window_start?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
causal_window_end?: string
/**
 * Гипотезы с основаниями.
 */
hypotheses: Hypothesis[]
/**
 * Нехватка сведений (FR-58, FR-143).
 * 
 * Items: Недостающее сведение.
 */
missing_information: ("tool_unknown" | "cycle_end_time_unknown" | "no_observation_after_operation" | "no_observation_before_operation" | "operator_unknown" | "equipment_log_missing" | "other")[]
/**
 * Категоричный ли вывод; при недостатке сведений — false (кейс §5.1 «Неопределённость»).
 */
conclusion_is_categorical: boolean
}
/**
 * Гипотеза причины — только предположение (третий статус кейса §2.3).
 */
export interface Hypothesis {
/**
 * Категория причины.
 */
category: ("incoming" | "equipment" | "performer" | "handling" | "assembly" | "documentation" | "not_established")
/**
 * Доля в базисных пунктах: 0…10000 (10000 = 1,0). Float в контрактах запрещён (AD-4).
 */
confidence_bp?: number
/**
 * Доводы «за».
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
supporting_event_ids: string[]
/**
 * Доводы «против».
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
contradicting_event_ids?: string[]
}
/**
 * Гипотеза причины записана человеком — технолог добавил гипотезу с основаниями и альтернативами; две ветки — «почему возник» и «почему пропустили» (FR-59).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentHypothesisRecordedV1".
 */
export interface IncidentHypothesisRecordedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id: string
/**
 * Несоответствия.
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
nc_ids?: string[]
/**
 * Ветка.
 */
branch: ("why_made" | "why_missed")
/**
 * Категория.
 */
category: ("incoming" | "equipment" | "performer" | "handling" | "assembly" | "documentation" | "not_established")
/**
 * Формулировка гипотезы.
 */
statement: string
/**
 * Основания.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
supporting_event_ids?: string[]
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
hypothesis_id?: string
/**
 * Что записано: предложена гипотеза / гипотеза отклонена с основанием (вывод системы не переписывается, FR-59). Нет поля — proposed.
 */
verdict?: ("proposed" | "rejected")
reason?: Reason26
}
/**
 * Причина действия: код и текст.
 */
export interface Reason26 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Инцидент закрыт — итог: сколько было в области, сколько подтверждено, сколько исключено; показатель сокращения области (FR-9).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentIncidentClosedV1".
 */
export interface IncidentIncidentClosedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id: string
/**
 * Размер области при создании.
 */
initial_size: number
/**
 * Подтверждено.
 */
confirmed: number
/**
 * Исключено.
 */
excluded: number
/**
 * Итог.
 */
summary?: string
/**
 * Что закрыто: область риска (risk_scope, по умолчанию) или расследование целиком (investigation — обе причины отвечены, эффективность мер проверена).
 */
scope?: ("risk_scope" | "investigation")
}
/**
 * Инцидент открыт — связанная группа сигналов и несоответствий с общей предполагаемой причиной; вычисляет межизделийная стадия функциями analysis (AD-42).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentIncidentOpenedV1".
 */
export interface IncidentIncidentOpenedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id: string
/**
 * Общий фактор.
 */
common_factor: ("equipment" | "tool" | "fixture" | "program" | "lot" | "heat" | "charge" | "operator" | "time_window" | "supplier" | "other")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
factor_ref?: string
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key?: string
/**
 * Записи, открывшие инцидент.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
trigger_event_ids: string[]
}
/**
 * Изделие в области проверено — контролёр проверил изделие из области: подтверждено / исключено; «под подозрением» стало «красным» или «зелёным» (FR-62).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentItemAssessedV1".
 */
export interface IncidentItemAssessedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id: string
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
item_id: string
/**
 * Итог проверки.
 */
assessment: ("confirmed" | "excluded")
/**
 * Чем проверили.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
evidence_event_ids: string[]
}
/**
 * Запрошено измерение для проверки гипотезы — технолог просит измерить (контрольный образец, рентген, замер режима); задачу исполнителю ставит notifications по этой записи; вывод о причине не меняется до решения человека (FR-59, FR-135).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentMeasurementRequestedV1".
 */
export interface IncidentMeasurementRequestedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id: string
/**
 * Несоответствия.
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
nc_ids?: string[]
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
hypothesis_id: string
/**
 * Что измерить.
 */
what: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
assignee_id?: string
}
/**
 * Статус изделия в инциденте изменён — адресованная запись межизделийной стадии изделию: что известно и что делать (FR-62); распространяется вверх по дереву сборки (AD-42).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentMembershipChangedV1".
 */
export interface IncidentMembershipChangedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id: string
/**
 * Версия области.
 */
scope_version: number
/**
 * Что известно об изделии в этом инциденте.
 */
status: ("confirmed" | "suspect" | "excluded" | "unknown")
/**
 * Что делать с изделием.
 */
action: ("observe" | "check" | "block" | "release")
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
via_assembly_of?: string
}
/**
 * Ошибка исполнителя подтверждена — юридически значимый факт: только уполномоченным после письменного объяснения работника (ТК РФ ст. 247); система взысканий не рассчитывает (FR-59).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentOperatorErrorConfirmedV1".
 */
export interface IncidentOperatorErrorConfirmedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id: string
/**
 * Несоответствия.
 * 
 * @minItems 1
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
nc_ids: [string, ...(string)[]]
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
operator_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
explanation_document_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
commission_document_id?: string
}
/**
 * Новая версия области риска вычислена — область риска по общему фактору в окне от последней подтверждённо годной детали до обнаружения; расширяется консервативно; разбивка по местонахождению (FR-61).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentScopeComputedV1".
 */
export interface IncidentScopeComputedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id: string
/**
 * Версия области.
 */
scope_version: number
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
window_start: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
window_end: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
last_known_good_event_id?: string
/**
 * Добавленные изделия.
 * 
 * Items: Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
added_item_ids?: string[]
/**
 * Размер области после версии.
 */
size: number
breakdown: ScopeBreakdown
/**
 * Основания — `event_id` записей, на которые опирается запись.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
basis: string[]
/**
 * Что дало версию: правило системы (вычислена или расширена правилом) / решение человека «расширить» / решение человека «сузить». Нет поля — computed.
 */
change?: ("computed" | "expanded" | "narrowed")
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
decision_event_id?: string
/**
 * Исключённые изделия — только по решению человека с основанием (FR-61, AD-27); правило системы изделий не исключает.
 * 
 * Items: Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
removed_item_ids?: string[]
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
last_known_good_item_id?: string
}
/**
 * Разбивка: в производстве / ушли дальше / собраны / отгружены (FR-61).
 */
export interface ScopeBreakdown {
/**
 * В производстве.
 */
in_production: number
/**
 * Ушли дальше.
 */
moved_on: number
/**
 * Собраны.
 */
assembled: number
/**
 * Отгружены.
 */
shipped: number
}
/**
 * Область риска расширена человеком — осторожный шаг: человек добавляет изделия с основанием (FR-61).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentScopeExpandedV1".
 */
export interface IncidentScopeExpandedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id: string
/**
 * Добавляемые изделия.
 * 
 * @minItems 1
 * 
 * Items: Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
item_ids: [string, ...(string)[]]
reason: Reason27
}
/**
 * Причина действия: код и текст.
 */
export interface Reason27 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Область риска сужена — сужение — только человеком с основанием (доказательство, автор, время); каждая правка — новая версия; незаметных сужений нет (FR-61, FR-144).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentScopeNarrowedV1".
 */
export interface IncidentScopeNarrowedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id: string
/**
 * Исключаемые изделия.
 * 
 * @minItems 1
 * 
 * Items: Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
item_ids: [string, ...(string)[]]
/**
 * Доказательства сужения.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
evidence_event_ids: string[]
/**
 * Снять блок с исключённых (как настроено, FR-62).
 */
release_containment?: boolean
reason: Reason28
}
/**
 * Причина действия: код и текст.
 */
export interface Reason28 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Предложение передано ответственному — решение руководителя (UJ-1): задачу ответственному ставит notifications; система сама ничего не меняет (FR-63).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentSuggestionForwardedV1".
 */
export interface IncidentSuggestionForwardedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
suggestion_id: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
responsible_id: string
/**
 * Роль ответственного: мастер участка — по персоналу, технолог — если нужна новая версия процесса.
 */
responsible_role?: string
/**
 * Заголовок предложения — для задачи ответственному.
 */
title?: string
/**
 * Пояснение руководителя.
 */
note?: string
}
/**
 * Предложение записано — предложение генератора (ограничение линии, область риска, кандидат в правило, адаптация VisionQC, карта дефицита данных) с основаниями и ответственным; ничего не применяет само; дальше — решения людей incident.suggestion.forwarded и incident.suggestion.resolved. Недетерминированный генератор — источник факта (AD-3, FR-63).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentSuggestionRecordedV1".
 */
export interface IncidentSuggestionRecordedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
suggestion_id: string
/**
 * Генератор (порт + адаптер).
 */
generator: string
/**
 * Вид предложения.
 */
kind: ("bottleneck" | "risk_scope" | "reaction_rule_candidate" | "analyzer_adaptation" | "data_deficit" | "other")
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
responsible_id?: string
/**
 * Суть предложения.
 */
statement: string
/**
 * Основания — `event_id` записей, на которые опирается запись.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
basis: string[]
/**
 * Короткий заголовок предложения для списка.
 */
title?: string
/**
 * Роль ответственного (мастер участка, технолог, руководитель производства…), если человек не назначен.
 */
responsible_role?: string
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
incident_id?: string
/**
 * Оценка эффекта словами с единицами («детали ждут в среднем 12 мин — около 40 деталей в смену»; «сузило бы область в среднем с 13 до 4 деталей»).
 */
estimate?: string
/**
 * Вид недостающих сведений (для предложения карты дефицита данных, FR-143) — значение перечисления missing_information разбора.
 */
missing_kind?: string
/**
 * Ключ повторения: генератор не записывает второе предложение с тем же ключом, пока первое не решено.
 */
dedup_key?: string
}
/**
 * Решение по предложению: принято в работу или отклонено — с основанием. «Принято» не применяет ничего само: изменение нормы — новая версия процесса через кворум, мера — incident.action.assigned (FR-63, FR-64).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IncidentSuggestionResolvedV1".
 */
export interface IncidentSuggestionResolvedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
suggestion_id: string
/**
 * Итог: принято в работу / отклонено.
 */
resolution: ("accepted" | "rejected")
reason: Reason29
}
/**
 * Причина действия: код и текст.
 */
export interface Reason29 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Флаг аномалии входа — время из будущего, расхождение часов источника выше порога, нарушение последовательности, неизвестное значение перечисления `UNKNOWN(значение)` — принято с флагом (FR-29, FR-33, AD-5).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IngestAnomalyFlaggedV1".
 */
export interface IngestAnomalyFlaggedV1 {
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
subject_event_id: string
/**
 * Флаг.
 */
flag: ("future_timestamp" | "clock_skew" | "sequence_violation" | "unknown_enum_value" | "unknown_defect_type" | "late_write")
/**
 * Поле (для неизвестного значения).
 */
field?: string
/**
 * Исходное значение: `UNKNOWN(значение)`.
 */
raw_value?: string
/**
 * Расхождение часов, мс.
 */
skew_ms?: number
}
/**
 * Импорт журнала завершён — импорт Excel/CSV (журнал сварки, бумажный цех) через адаптер источника с проверкой входов, дублей и привязки; события получают пометку «импорт» (FR-141).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IngestImportCompletedV1".
 */
export interface IngestImportCompletedV1 {
/**
 * Идентификатор источника событий: устройство, шлюз, терминал, партнёр (`partner:‹код›`); в прогоне сценария — `‹run_id›/‹источник›` (AD-38).
 */
source_id: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
file_digest: string
/**
 * Имя файла.
 */
file_name?: string
/**
 * Строк всего.
 */
rows_total: number
/**
 * Принято.
 */
rows_accepted: number
/**
 * Дубли.
 */
rows_duplicate: number
/**
 * Отклонено.
 */
rows_rejected: number
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
imported_by?: string
}
/**
 * Сообщение помещено в карантин — содержимое — вне журнала (MaterialStore, зашифровано), в журнале — факт помещения: источник, номер, отпечаток, код причины (AD-2, FR-30).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IngestMessageQuarantinedV1".
 */
export interface IngestMessageQuarantinedV1 {
/**
 * Идентификатор источника событий: устройство, шлюз, терминал, партнёр (`partner:‹код›`); в прогоне сценария — `‹run_id›/‹источник›` (AD-38).
 */
source_id: string
/**
 * Номер у источника, если прочитан.
 */
source_seq?: number
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
event_id?: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
fingerprint: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
material_address: string
/**
 * Код причины из contracts/errors.yaml.
 */
problem_code: string
/**
 * Подробности ошибки.
 */
detail?: string
}
/**
 * Сообщение из карантина переобработано — администратор повторно обработал сообщение (например, после появления повышателя версии) (FR-30).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IngestMessageReprocessedV1".
 */
export interface IngestMessageReprocessedV1 {
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
quarantine_event_id: string
/**
 * Итог.
 */
outcome: ("accepted" | "still_invalid" | "discarded")
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
accepted_event_id?: string
reason?: Reason30
}
/**
 * Причина действия: код и текст.
 */
export interface Reason30 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Возможна потеря данных источника — разрыв `source_seq` не закрыт досылкой за окно ожидания — реакция планировщика; досылка в окне её отменяет (AD-7).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "IngestSourceLossSuspectedV1".
 */
export interface IngestSourceLossSuspectedV1 {
/**
 * Идентификатор источника событий: устройство, шлюз, терминал, партнёр (`partner:‹код›`); в прогоне сценария — `‹run_id›/‹источник›` (AD-38).
 */
source_id: string
/**
 * Нет с номера.
 */
missing_from_seq: number
/**
 * По номер.
 */
missing_to_seq: number
}
/**
 * Режим точки контроля изменён — точка контроля работает / ухудшена / только ручной контроль / выведена; результаты в ручном режиме получают источник «человек».
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "InspectionPointModeChangedV1".
 */
export interface InspectionPointModeChangedV1 {
/**
 * Точка контроля (КТ).
 */
inspection_point: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id?: string
/**
 * Новый режим.
 */
mode: ("operational" | "degraded" | "manual_only" | "out_of_service")
reason?: Reason31
}
/**
 * Причина действия: код и текст.
 */
export interface Reason31 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Результат контроля записан — единый тип результата любого метода (камера, КИМ, рентген, стенд, человек) с тремя исходами; «не обнаружено» при плохом наблюдении — не годность (FR-36, кейс §4.5). Имя типа результат не кодирует (AD-23).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "InspectionResultRecordedV1".
 */
export interface InspectionResultRecordedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
observation_id?: string
/**
 * Метод контроля (FR-14, FR-27): камера, КИМ, рентген, УЗК, капиллярный, течеискатель, момент затяжки, визуальный человеком, документы поставщика, лаборатория, иное.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "inspection_method".
 */
method: ("camera" | "cmm" | "radiography" | "ultrasonic" | "penetrant" | "leak_test" | "torque" | "visual_human" | "supplier_documents" | "laboratory" | "other")
/**
 * Фаза контроля: вход / до операции / после операции / перед закрытием зоны / сборка / испытание / окончательный.
 */
phase: ("incoming" | "before_operation" | "after_operation" | "before_zone_closure" | "assembly" | "test" | "final" | "other")
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key?: string
/**
 * Точка контроля (КТ) процесса, например `KT-3`.
 */
inspection_point?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
lot_id?: string
/**
 * Исход результата контроля (кейс §4.4, FR-36): признаки дефекта обнаружены / не обнаружены / оценка невозможна.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "inspection_outcome".
 */
outcome: ("defect_indicated" | "no_defect_indicated" | "unable_to_assess")
/**
 * Почему оценка невозможна (если исход `unable_to_assess`).
 */
unable_reason?: ("poor_image" | "zone_occluded" | "carrier_unreadable" | "analyzer_failure" | "processing_aborted" | "not_measured" | "document_missing" | "test_invalid" | "other")
/**
 * Состояние обработки у источника (OPC UA Machine Vision): выполнено / прервано / сбой. «Прервано» и «сбой» дают «оценка невозможна».
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "processing_state".
 */
processing_state: ("completed" | "aborted" | "failed")
/**
 * Зоны, которые охватило наблюдение.
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
zone_ids?: string[]
/**
 * Признаки дефектов; один сигнал может содержать несколько (FR-37).
 */
defects?: Defect[]
/**
 * Измерения «значение против допуска».
 */
measurements?: MeasurementResult[]
/**
 * Сравнение с состоянием зоны до операции.
 */
comparison_before?: ("was_before" | "appeared" | "changed" | "unchanged" | "not_comparable")
/**
 * Доля в базисных пунктах: 0…10000 (10000 = 1,0). Float в контрактах запрещён (AD-4).
 */
analyzer_confidence_bp?: number
/**
 * Доля в базисных пунктах: 0…10000 (10000 = 1,0). Float в контрактах запрещён (AD-4).
 */
observation_quality_bp?: number
/**
 * Ступени анализатора с версиями и уверенностью (FR-38).
 */
stages?: AnalyzerStage[]
/**
 * Рекомендация анализатора — только совет, решает карта реакций (FR-48).
 */
recommendation?: ("pass_to_next" | "manual_review" | "isolate" | "none")
/**
 * Ограничения наблюдения и метода.
 * 
 * Items: Ограничение наблюдения.
 */
limitations?: string[]
versions?: AnalyzerVersions3
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id?: string
/**
 * Контролёр или дефектоскопист, если контроль выполнял человек.
 */
inspector_id?: (string | null)
/**
 * Номер заключения (НК, протокол КИМ, протокол испытаний).
 */
conclusion_ref?: string
/**
 * Материалы; отсутствие обрабатывается корректно (кейс §5.4).
 */
evidence_refs?: EvidenceRef[]
}
/**
 * Вектор версий наблюдения (AD-29); для человека-контролёра — версия карты контроля и приложения.
 */
export interface AnalyzerVersions3 {
/**
 * Ревизия изделия (КД).
 */
item_revision?: string
/**
 * Карта контроля с версией: `‹id›@‹версия›`.
 */
recipe_ref: string
/**
 * Конфигурация камеры и света.
 */
camera_config?: string
/**
 * Калибровка.
 */
calibration?: string
/**
 * Версия анализатора (внешнего модуля).
 */
analyzer_version: string
/**
 * Профиль порогов.
 */
threshold_profile?: string
/**
 * Версия контракта данных источника.
 */
contract_version: string
/**
 * Версия приложения источника.
 */
app_version?: string
}
/**
 * Компонент включён в сборку — компонент (экземпляр или партия) установлен в сборку или соединён сваркой; история сборки включает историю компонентов (FR-45). Сканируется до установки.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ItemAssemblyRecordedV1".
 */
export interface ItemAssemblyRecordedV1 {
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
assembly_item_id: string
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
component_item_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
component_lot_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
component_type_id: string
/**
 * Количество (для партионных компонентов).
 */
quantity?: number
/**
 * Позиция по спецификации или карте сборки.
 */
position?: string
/**
 * Тип носителя идентификатора (AD-16): DPM DataMatrix / бирка с QR / тара и ячейка / сопроводительная карта / контекст поста / ручной ввод / внутренний ID системы.
 */
binding_method: ("dpm_datamatrix" | "tag_qr" | "container_cell" | "route_card" | "post_context" | "manual_entry" | "internal_id")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id?: string
}
/**
 * Носитель идентификатора нанесён — на изделие нанесён носитель (DPM DataMatrix, бирка с QR, тара и ячейка); связь «носитель → изделие» (AD-16).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ItemCarrierAppliedV1".
 */
export interface ItemCarrierAppliedV1 {
/**
 * Тип носителя идентификатора (AD-16): DPM DataMatrix / бирка с QR / тара и ячейка / сопроводительная карта / контекст поста / ручной ввод / внутренний ID системы.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "carrier_type".
 */
carrier_type: ("dpm_datamatrix" | "tag_qr" | "container_cell" | "route_card" | "post_context" | "manual_entry" | "internal_id")
/**
 * Значение носителя.
 */
value: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
zone_id?: string
/**
 * Временный носитель (бирка до DPM); не снятый временный носитель — задача контроля полноты.
 */
is_temporary: boolean
/**
 * Значение заменяемого носителя, если это перемаркировка; null — первичная маркировка.
 */
replaces_value?: (string | null)
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
lot_id?: string
}
/**
 * Носитель снят — носитель снят с изделия (временная бирка снята, код снят мехобработкой); разрешение носителя учитывает «снят» на `occurred_at` (AD-41).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ItemCarrierRemovedV1".
 */
export interface ItemCarrierRemovedV1 {
/**
 * Тип носителя идентификатора (AD-16): DPM DataMatrix / бирка с QR / тара и ячейка / сопроводительная карта / контекст поста / ручной ввод / внутренний ID системы.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "carrier_type".
 */
carrier_type: ("dpm_datamatrix" | "tag_qr" | "container_cell" | "route_card" | "post_context" | "manual_entry" | "internal_id")
/**
 * Значение снятого носителя.
 */
value: string
reason?: Reason32
}
/**
 * Причина действия: код и текст.
 */
export interface Reason32 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Носитель проверен считыванием — результат считывания носителя: прочитан / нечитаем / не совпал; нечитаемый носитель — задача перемаркировки (AD-16).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ItemCarrierVerifiedV1".
 */
export interface ItemCarrierVerifiedV1 {
/**
 * Тип носителя идентификатора (AD-16): DPM DataMatrix / бирка с QR / тара и ячейка / сопроводительная карта / контекст поста / ручной ввод / внутренний ID системы.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "carrier_type".
 */
carrier_type: ("dpm_datamatrix" | "tag_qr" | "container_cell" | "route_card" | "post_context" | "manual_entry" | "internal_id")
/**
 * Считанное значение (пусто, если не прочитан).
 */
value?: string
/**
 * Исход считывания.
 */
read_outcome: ("readable" | "unreadable" | "mismatch")
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key?: string
}
/**
 * Идентификация подтверждена человеком — повторная идентификация уполномоченным с подписью снимает «идентификацию под сомнением» (AD-16).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ItemIdentificationConfirmedV1".
 */
export interface ItemIdentificationConfirmedV1 {
/**
 * Тип носителя идентификатора (AD-16): DPM DataMatrix / бирка с QR / тара и ячейка / сопроводительная карта / контекст поста / ручной ввод / внутренний ID системы.
 */
method: ("dpm_datamatrix" | "tag_qr" | "container_cell" | "route_card" | "post_context" | "manual_entry" | "internal_id")
/**
 * Прочитанное значение носителя.
 */
carrier_value?: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
questioned_event_id: string
}
/**
 * Идентификация изделия под сомнением — потеря или неоднозначность идентификации: изделие изолируется до повторной идентификации человеком с подписью (AD-16).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ItemIdentificationQuestionedV1".
 */
export interface ItemIdentificationQuestionedV1 {
/**
 * Что случилось с идентификацией.
 */
cause_kind: ("carrier_unreadable" | "carrier_mismatch" | "ambiguous_binding" | "carrier_missing")
/**
 * Кандидаты при неоднозначности.
 * 
 * Items: Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
candidates?: string[]
/**
 * Основания — `event_id` записей, на которые опирается запись.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
basis: string[]
}
/**
 * Вмешательство закрыто — установлено обратно, зона проверена повторно, подпись контролёра; приёмка снова возможна (FR-21, ЗТ-В).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ItemInterventionClosedV1".
 */
export interface ItemInterventionClosedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
intervention_id: string
/**
 * Результаты повторной проверки зон (`inspection.result.recorded`).
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
recheck_event_ids: string[]
/**
 * Вскрывали после испытания — нужно повторное испытание.
 */
retest_required?: boolean
}
/**
 * Вмешательство в собранное изделие открыто — разборка собранного или принятого изделия: что, зона, кто, зачем; пока открыто — приёмка и отгрузка запрещены, прежние результаты зоны «устарели» (FR-21).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ItemInterventionOpenedV1".
 */
export interface ItemInterventionOpenedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
intervention_id: string
/**
 * Затронутые зоны.
 * 
 * @minItems 1
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
zone_ids: [string, ...(string)[]]
/**
 * Что снято (крышка, крепёж).
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
removed_components?: string[]
purpose: Reason33
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
qc_consent_by?: string
}
/**
 * Причина действия: код и текст.
 */
export interface Reason33 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Изделие зарегистрировано и запущено в работу — у изделия появляется внутренний ID и паспорт; закрепляются версия процесса и нормативного слоя (AD-16, AD-17). ID из метки не выводится.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ItemItemRegisteredV1".
 */
export interface ItemItemRegisteredV1 {
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
item_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
item_type_id: string
/**
 * Ревизия КД и ТП на момент запуска.
 */
item_revision: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
order_id?: string
/**
 * Партии материала и покупных, из которых изготавливается изделие (корни генеалогии).
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
lot_ids?: string[]
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
process_version_hash: string
/**
 * Ревизия нормативного слоя, закреплённая при запуске: план контроля, карта реакций, классификатор, шаблоны.
 */
normative_rev: string
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
entry_step_key?: string
/**
 * Изделие — сборочная единица.
 */
is_assembly?: boolean
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
split_from?: string
}
/**
 * Изделие предъявлено — мастер предъявляет изделие ОТК или представителю заказчика на точке предъявления; номер предъявления растёт, повторное требует подписи выше (FR-19).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ItemPresentationRecordedV1".
 */
export interface ItemPresentationRecordedV1 {
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key: string
/**
 * Номер предъявления: 1 — первое, 2 — повторное и т. д.
 */
presentation_no: number
/**
 * Кому предъявлено.
 */
presented_to: ("qc" | "customer_representative")
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
presented_by: string
}
/**
 * Изделие сдано на склад готовой продукции — кладовщик принял годное изделие на склад; это исполнение закрывающей точки выпуска, следом — «выпуск годного» в 1С (FR-44).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ItemReleaseRecordedV1".
 */
export interface ItemReleaseRecordedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
warehouse_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
concession_id?: string
/**
 * Изделие принято после переделки.
 */
after_rework: boolean
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
received_by: string
}
/**
 * Ключ-якорь уничтожен — закрытый ключ-якорь уничтожен сразу после подписи генезиса (AD-33).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "JournalAnchorDestroyedV1".
 */
export interface JournalAnchorDestroyedV1 {
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
anchor_fingerprint: string
}
/**
 * Сменилась сборка доменного пакета — после смены сборки воркер пересматривает слоты с пометкой «пересмотрено из-за обновления» (AD-9).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "JournalBuildChangedV1".
 */
export interface JournalBuildChangedV1 {
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
domain_build: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
previous_build?: string
}
/**
 * Контрольная точка хранителя — хранитель принял головы цепочек: `seq`, звено, диапазон `committed_at`, отпечаток предыдущей точки; пакет DSSE `hybrid` у хранителя (AD-8).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "JournalCheckpointRecordedV1".
 */
export interface JournalCheckpointRecordedV1 {
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
checkpoint_digest: string
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 */
main_seq: number
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
main_link: string
/**
 * Голова цепочки CA.
 */
ca_seq?: number
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
ca_link?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
keeper_time: string
}
/**
 * Блок генезиса — заголовок блока генезиса `seq` 1…k: код предприятия, криптопрофили, формат цепочки, отпечатки якоря и ключей; подписан ключом-якорем `hybrid` (AD-33).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "JournalGenesisRecordedV1".
 */
export interface JournalGenesisRecordedV1 {
/**
 * Код предприятия.
 */
enterprise_code: string
/**
 * Версия формата цепочки.
 */
chain_format_version: number
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
anchor_fingerprint: string
/**
 * Число записей блока генезиса.
 */
block_size: number
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
normative_version_hash?: string
/**
 * Открытые ключи якоря (`hybrid`: ГОСТ и ML-DSA-65) — ими проверяются подписи всех записей блока; их общий отпечаток — `anchor_fingerprint`, закреплённый в `trust-anchors` вне системы (AD-33). Закрытый ключ-якорь уничтожен (`journal.anchor.destroyed`).
 * 
 * @minItems 1
 * @maxItems 4
 */
anchor_keys?: [GenesisAnchorKey]|[GenesisAnchorKey, GenesisAnchorKey]|[GenesisAnchorKey, GenesisAnchorKey, GenesisAnchorKey]|[GenesisAnchorKey, GenesisAnchorKey, GenesisAnchorKey, GenesisAnchorKey]
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
block_digest?: string
}
export interface GenesisAnchorKey {
/**
 * Ссылка на ключ подписанта: `key_id@версия` (AD-10).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "key_ref".
 */
key_ref: string
/**
 * Профиль ключа.
 */
profile_id: ("gost" | "pq")
/**
 * Открытый ключ в base64 (кодирование — contracts/crypto/README.md).
 */
public_key_b64: string
}
/**
 * Акт восстановления — журнал из копии старше контрольной точки принимается только при подписанном акте (администратор безопасности + Аудитор ИБ) с диапазоном потерянных номеров (AD-34, описание).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "JournalRestoreRecordedV1".
 */
export interface JournalRestoreRecordedV1 {
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 */
lost_from_seq: number
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 */
lost_to_seq: number
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id: string
reason: Reason34
}
/**
 * Причина действия: код и текст.
 */
export interface Reason34 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Открыт новый сегмент цепочки — смена хеш-функции — новый сегмент: связь головы старого с перехешированием (AD-32).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "JournalSegmentOpenedV1".
 */
export interface JournalSegmentOpenedV1 {
/**
 * Новая версия формата цепочки.
 */
chain_format_version: number
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 */
previous_head_seq: number
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
previous_head_link: string
}
/**
 * Криптопрофиль зарегистрирован — реестр профилей: какой профиль обязателен для класса объекта с какого момента (AD-10, AD-32).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "KeyProfileRegisteredV1".
 */
export interface KeyProfileRegisteredV1 {
/**
 * Профиль.
 */
profile_id: ("gost" | "pq" | "hybrid")
/**
 * Классы пакетов, для которых профиль обязателен.
 * 
 * @minItems 1
 * 
 * Items: Класс подписанного пакета (contracts/crypto/payload-classes.yaml): как code, но допускает дефис — `event`, `document-signature`, `key-act` (Д-66).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "class".
 */
object_classes: [string, ...(string)[]]
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 */
effective_from_seq?: number
}
/**
 * Ключ зарегистрирован актом — акт регистрации ключа человека, устройства или системы: открытый ключ, профиль, допустимые классы пакетов, доказательство владения, подтверждение субъекта, маршрут подписей (AD-11). Ротация — регистрация с `rotates`.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "KeyRegistrationRecordedV1".
 */
export interface KeyRegistrationRecordedV1 {
/**
 * Ссылка на ключ подписанта: `key_id@версия` (AD-10).
 */
key_ref: string
/**
 * Чей ключ.
 */
subject_kind: ("person" | "device" | "engine" | "gateway" | "enterprise_gateway" | "keeper" | "verifier" | "demo_persona" | "partner_root")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
subject_id: string
/**
 * Профиль ключа.
 */
profile_id: ("gost" | "pq")
/**
 * Алгоритм.
 */
algorithm: ("gost3410_2012_256_paramset_a" | "ml_dsa_65")
/**
 * Открытый ключ в base64 (кодирование — contracts/crypto/README.md).
 */
public_key_b64: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
fingerprint: string
/**
 * Допустимые классы пакетов для ключа.
 * 
 * @minItems 1
 * 
 * Items: Класс подписанного пакета (contracts/crypto/payload-classes.yaml): как code, но допускает дефис — `event`, `document-signature`, `key-act` (Д-66).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "class".
 */
payload_classes: [string, ...(string)[]]
/**
 * Ссылка на ключ подписанта: `key_id@версия` (AD-10).
 */
rotates?: string
/**
 * Подпись нового ключа над актом (доказательство владения).
 */
proof_of_possession_b64?: string
/**
 * Подтверждение субъекта: подпись действующим ключом при ротации / бумажная расписка с отпечатком при первичной выдаче.
 */
subject_confirmation: ("rotation_signature" | "paper_receipt" | "genesis")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_from: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_until?: string
/**
 * Класс хранения ключа (AD-11, AD-14, Д-72): hardware_token — физический ключ через агент токена (эталон); software_browser — ключ в браузере, зашифрованный под PIN (умышленно сниженный порог). Для ключей людей; у системных ключей и устройств отсутствует. Виден в подписи, аудите и отчёте верификатора.
 */
key_storage?: ("hardware_token" | "software_browser")
/**
 * Разновидность хранения ключа в браузере: extension — в расширении (окно подтверждения — страница расширения); page — в хранилище страницы (планшет, телефон; сводку показывает код, отданный сервером, — доверенного отображения нет).
 */
storage_variant?: ("extension" | "page")
}
/**
 * Ключ отозван — отзыв несёт «скомпрометирован с X» (X может быть раньше отзыва) и порождает защитную реакцию для решений, подписанных после X (AD-11).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "KeyRevocationRecordedV1".
 */
export interface KeyRevocationRecordedV1 {
/**
 * Ссылка на ключ подписанта: `key_id@версия` (AD-10).
 */
key_ref: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
compromised_since?: string
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 */
compromised_since_seq?: number
reason: Reason35
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id?: string
}
/**
 * Причина действия: код и текст.
 */
export interface Reason35 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Сменный рапорт записан — подпись уровня 3 над корнем дерева Меркла отпечатков подписей смены из локального журнала агента токена и итог сверки с журналом сервера; расхождение — тревога agent_journal_mismatch (FR-66, FR-81, AD-12, AD-14).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "KeyShiftReportRecordedV1".
 */
export interface KeyShiftReportRecordedV1 {
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
person_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
shift_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
workplace_id?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
window_from: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
window_to: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
merkle_root: string
/**
 * Число подписей по рапорту агента.
 */
leaf_count: number
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
server_merkle_root?: string
/**
 * Число подписей по журналу сервера.
 */
server_leaf_count?: number
/**
 * Рапорт агента совпал с журналом сервера.
 */
matches: boolean
/**
 * Типы записей, по которым расходятся счётчики.
 * 
 * Items: Тип события: семейство.сущность.действие — три сегмента латиницей (AD-40).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "event_type".
 */
mismatch_types?: string[]
/**
 * Подписанный агентом конверт DSSE класса shift-report (contracts/crypto/dsse-envelope.schema.json) — как есть.
 */
signed_report: {

}
}
/**
 * Материал помещён в хранилище — фото, видео, скан, протокол, иллюстрация — по адресу H(байты), зашифровано; в журнале — адрес и метаданные: источник, время, изделие, точка контроля, происхождение (FR-102, AD-23).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "MaterialObjectStoredV1".
 */
export interface MaterialObjectStoredV1 {
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
material_address: string
/**
 * MIME-тип.
 */
media_type: string
/**
 * Размер.
 */
size_bytes: number
/**
 * Вид.
 */
kind: ("photo" | "video" | "illustration" | "protocol" | "log_excerpt" | "scan" | "quarantine_payload" | "other")
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
captured_at?: string
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
item_id?: string
/**
 * Точка контроля.
 */
inspection_point?: string
/**
 * Иллюстрация из открытого набора (FR-102).
 */
is_illustration: boolean
/**
 * Источник, лицензия, автор; для иллюстраций — «не относится к изделию».
 */
provenance_note?: string
}
/**
 * Блокировка передаётся в MES — заблокированное изделие или партия не должны получить следующую операцию — блок уходит в MES (FR-93).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "MesHoldRequestedV1".
 */
export interface MesHoldRequestedV1 {
/**
 * Бизнес-ключ.
 */
business_key: string
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
item_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
lot_id?: string
/**
 * true — заблокировать, false — снять.
 */
hold: boolean
}
/**
 * Ответ MES на блокировку — квитанция MES.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "MesHoldRespondedV1".
 */
export interface MesHoldRespondedV1 {
/**
 * Бизнес-ключ.
 */
business_key: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
request_event_id: string
/**
 * Итог.
 */
outcome: ("accepted" | "duplicate" | "rejected")
/**
 * Код ошибки.
 */
error_code?: string
}
/**
 * Получено задание MES — сменное задание или наряд из MES (подмножество B2MML-JSON, AD-18) — без повторного ручного ввода (FR-93).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "MesJobReceivedV1".
 */
export interface MesJobReceivedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
job_id: string
/**
 * Номер в MES.
 */
external_number: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
order_id?: string
/**
 * Операция.
 */
operation_code: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
station_id?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
planned_start?: string
}
/**
 * Версия процесса введена в действие — после закрытия маршрута кворума; действует только для изделий, запущенных после (PRD §11.9, FR-23).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "NormativeVersionActivatedV1".
 */
export interface NormativeVersionActivatedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
version_id: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
process_version_hash: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
route_closed_event_id: string
}
/**
 * Черновик версии процесса сохранён — технолог сохранил BPMN из редактора; загрузчик проверил его (FR-13); дальше — отправка на утверждение кворумом `normative.version.submitted` (FR-22, FR-25). Черновик не действует до введения в действие.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "NormativeVersionDraftedV1".
 */
export interface NormativeVersionDraftedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
version_id: string
/**
 * Метка версии для людей.
 */
label: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
base_version_id?: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
process_version_hash: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
bundle_digest?: string
/**
 * Номер сохранения черновика: каждое сохранение — новая запись с номером на единицу больше.
 */
revision?: number
}
/**
 * Стартовая версия нормативного слоя загружена — seed при первом запуске: BPMN как загружен, карта реакций, классификатор, шаблоны, политика; подписи кворума — ключами генезиса (FR-10, AD-33).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "NormativeVersionLoadedV1".
 */
export interface NormativeVersionLoadedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
version_id: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
process_version_hash: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
bundle_digest: string
/**
 * Состав пакета.
 * 
 * @minItems 1
 */
components: [NormativeFile, ...(NormativeFile)[]]
}
/**
 * Файл нормативного слоя.
 */
export interface NormativeFile {
/**
 * Путь в /normative.
 */
path: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
digest: string
}
/**
 * Версия выведена — версия больше не назначается новым изделиям; изделия прежних версий показываются на узлах своего `step_key` (AD-17).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "NormativeVersionRetiredV1".
 */
export interface NormativeVersionRetiredV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
version_id: string
reason: Reason36
}
/**
 * Причина действия: код и текст.
 */
export interface Reason36 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Версия отправлена на утверждение — черновик технолога → «на утверждении»: лист утверждения с читаемой разницей (FR-22…24).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "NormativeVersionSubmittedV1".
 */
export interface NormativeVersionSubmittedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
version_id: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
process_version_hash: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
bundle_digest: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
base_version_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
approval_document_id: string
}
/**
 * Срок снят — обязательство выполнено или больше не действует.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ObligationDueClearedV1".
 */
export interface ObligationDueClearedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
obligation_id: string
/**
 * Почему снят.
 */
cause: ("fulfilled" | "withdrawn" | "superseded")
}
/**
 * Наступил срок — планировщик по проекции сроков: `event_id` = UUIDv5(`obligation_id`, `due_at`) — повтор не дублирует (AD-4).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ObligationDueReachedV1".
 */
export interface ObligationDueReachedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
obligation_id: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
due_at: string
}
/**
 * Срок установлен — срок решения, точки предъявления, таймера BPMN: `due_at` — по производственному календарю и графику смен; эмитит только notifications (AD-4).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ObligationDueSetV1".
 */
export interface ObligationDueSetV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
obligation_id: string
/**
 * Вид срока.
 */
kind: ("nc_disposition" | "presentation_wait" | "bpmn_timer" | "isolation_move" | "task" | "containment_review" | "recheck" | "source_loss_window")
/**
 * Поток записи (AD-39): `item:‹item_id›`, `‹вид_объекта›:‹id›` или `global`.
 */
subject_ref: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
due_at: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
owner_role_id?: string
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key?: string
/**
 * Уровень эскалации, к которому относится срок: 1 — исходный срок; после «наступил срок» notifications переустанавливает срок следующего уровня того же обязательства (лестница эскалации, FR-57).
 */
level?: number
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
first_due_at?: string
/**
 * Поток записи (AD-39): `item:‹item_id›`, `‹вид_объекта›:‹id›` или `global`.
 */
waits_on?: string
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 */
basis?: string
/**
 * Что ждёт решения — для ленты тревог и блока «требует вашего внимания» (FR-8).
 */
title?: string
}
/**
 * Эскалация — просрочка с ценой задержки («просрочено 37 мин — стоят 18 изделий, 2 операции») (FR-57, FR-8).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ObligationEscalationRaisedV1".
 */
export interface ObligationEscalationRaisedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
obligation_id: string
/**
 * Уровень эскалации.
 */
level: number
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
escalate_to_role_id?: string
/**
 * На сколько просрочено, минут.
 */
overdue_minutes: number
/**
 * Изделий стоит.
 */
blocked_items?: number
/**
 * Операций стоит.
 */
blocked_operations?: number
}
/**
 * Достигнуто событие-сообщение процесса — токен прошёл промежуточное событие-сообщение BPMN (например, «В 1С: перемещение»); учётное действие формирует модуль erp (AD-18).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperationMessageThrownV1".
 */
export interface OperationMessageThrownV1 {
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key: string
/**
 * Сообщение BPMN (`bpmn:message/@id`).
 */
message_ref: string
/**
 * Учётное действие по свойству шага `ant:properties/@erpAction`; `return_from_defect` — «возврат из брака в производство» (решение Д-17).
 */
erp_action: ("accept_into_work" | "warehouse_transfer" | "scrap_transfer_rework" | "scrap_transfer_writeoff" | "scrap_transfer_reprocess" | "return_to_supplier" | "release" | "return_from_defect")
/**
 * Записи закрывающей точки, на которых основано действие.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
closing_basis: string[]
}
/**
 * Изделие принято на новом месте — принимающий подтвердил получение и осмотр; между цехами — закрывающая точка смены склада; в изолятор — физическое подтверждение изоляции (FR-16, FR-55).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperationMovementReceivedV1".
 */
export interface OperationMovementReceivedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
from_location_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
to_location_id: string
/**
 * Вид места назначения.
 */
destination_kind: ("station" | "workshop" | "warehouse" | "isolator" | "other")
/**
 * Осмотр при приёмке.
 */
inspection_on_receipt: ("no_damage" | "damage_found" | "not_inspected")
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
received_by: string
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key?: string
}
/**
 * Изделие отправлено — изделие отправлено между участками, цехами или на склад; идёт время ожидания (FR-16).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperationMovementSentV1".
 */
export interface OperationMovementSentV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
from_location_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
to_location_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
container_id?: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
sent_by: string
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key?: string
}
/**
 * Нарушено предусловие операции — перед операцией не выполнено условие на дату операции: квалификация, поверка, ревизия КД/ТП, срок годности материала, окно времени, проверка зоны, блок (FR-17, FR-20).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperationPreconditionFailedV1".
 */
export interface OperationPreconditionFailedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id?: string
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key: string
/**
 * Какое условие нарушено.
 */
precondition: ("qualification" | "equipment_verification" | "document_revision" | "material_expiry" | "time_window" | "zone_check" | "item_blocked" | "open_intervention" | "rework_limit" | "status_expired")
/**
 * Реакция по свойству шага: блокировка или нарушение с записью.
 */
mode: ("block" | "record_violation")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
subject_ref?: string
/**
 * Подробности для карточки.
 */
detail?: string
}
/**
 * Операция завершена — работа сделана или прервана; это не «годно». Длительность — со смыслом интервала и происхождением (кейс §4.6, FR-88).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperationRunFinishedV1".
 */
export interface OperationRunFinishedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id: string
/**
 * Итог выполнения: завершена / прервана.
 */
completion: ("completed" | "interrupted")
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
operation_started_at?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
operation_finished_at?: string
reported_duration?: Duration1
}
/**
 * Длительность, переданная источником (значение, единица, смысл интервала, происхождение).
 */
export interface Duration1 {
/**
 * Значение длительности.
 */
value: number
/**
 * Единица длительности.
 */
unit: ("ms" | "s" | "min" | "h")
/**
 * Смысл интервала: активная обработка / полное время на участке / иной интервал.
 */
meaning: ("active_processing" | "time_at_station" | "other")
/**
 * Пояснение, если смысл — `other`.
 */
meaning_note?: string
/**
 * Происхождение: передано источником / вычислено системой.
 */
origin: ("source_reported" | "system_computed")
}
/**
 * Интервал выполнения операции определён — по началу и концу выполнения вычислен интервал для привязки событий оборудования; публикуется межизделийной стадии (AD-29, AD-42).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperationRunIntervalResolvedV1".
 */
export interface OperationRunIntervalResolvedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
interval_start: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
interval_end?: string
/**
 * Откуда границы: переданы источником / вычислены системой.
 */
interval_origin: ("source_reported" | "system_computed")
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key?: string
/**
 * Шаг — специальный процесс по закреплённой версии процесса изделия (`ant:properties/@specialProcess`, FR-151): стадия относит выполнение к окнам нарушения режима.
 */
special_process?: boolean
}
/**
 * Операция приостановлена — перерыв внутри выполнения операции: наладка, сбой, ожидание, конец смены.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperationRunPausedV1".
 */
export interface OperationRunPausedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id: string
/**
 * Причина паузы.
 */
pause_reason: ("setup" | "failure" | "waiting" | "shift_end" | "other" | "unknown")
/**
 * Пояснение.
 */
note?: string
}
/**
 * Операция возобновлена — выполнение операции продолжено после паузы.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperationRunResumedV1".
 */
export interface OperationRunResumedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id: string
}
/**
 * Операция начата — начато выполнение операции над изделием; до начала проверяются предусловия (FR-17). Источник — терминал, MES или станок.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperationRunStartedV1".
 */
export interface OperationRunStartedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id: string
/**
 * Операция по маршрутной карте.
 */
operation_code: string
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
line_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
station_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
workplace_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id?: string
/**
 * Исполнитель; null — неизвестен (FR-123).
 */
operator_id: (string | null)
/**
 * Управляющая программа и её ревизия.
 */
program_ref?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
rework_of?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
group_id?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
operation_started_at?: string
}
/**
 * Действие исполнителя замечено (OperatorVision) — только гипотеза о действии: пропущен шаг, нарушен порядок, лишний шаг, инструмента нет; распознавания лиц нет, исполнитель — по входу на рабочее место; в показатели по исполнителям не идёт (FR-126).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperatorActionObservedV1".
 */
export interface OperatorActionObservedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
workplace_id?: string
/**
 * Шаг ТП.
 */
tp_step?: string
/**
 * Наблюдение.
 */
observation: ("step_missed" | "sequence_broken" | "extra_step" | "tool_missing" | "tool_returned" | "manual_intervention" | "other")
/**
 * Доля в базисных пунктах: 0…10000 (10000 = 1,0). Float в контрактах запрещён (AD-4).
 */
analyzer_confidence_bp?: number
/**
 * Доля в базисных пунктах: 0…10000 (10000 = 1,0). Float в контрактах запрещён (AD-4).
 */
observation_quality_bp?: number
versions?: AnalyzerVersions4
/**
 * Материалы.
 */
evidence_refs?: EvidenceRef[]
}
/**
 * Вектор версий.
 */
export interface AnalyzerVersions4 {
/**
 * Ревизия изделия (КД).
 */
item_revision?: string
/**
 * Карта контроля с версией: `‹id›@‹версия›`.
 */
recipe_ref: string
/**
 * Конфигурация камеры и света.
 */
camera_config?: string
/**
 * Калибровка.
 */
calibration?: string
/**
 * Версия анализатора (внешнего модуля).
 */
analyzer_version: string
/**
 * Профиль порогов.
 */
threshold_profile?: string
/**
 * Версия контракта данных источника.
 */
contract_version: string
/**
 * Версия приложения источника.
 */
app_version?: string
}
/**
 * Пропущена обязательная проверка — обязательной проверки нет — для этой точки «неизвестно»; изделие не пройдёт закрывающую точку, пока проверки нет; критическое действие исполнителя (FR-28, AD-28).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperatorCheckSkippedV1".
 */
export interface OperatorCheckSkippedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
workplace_id?: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
operator_id: string
/**
 * Какая проверка.
 */
inspection_point?: string
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key: string
reason?: Reason37
}
/**
 * Причина действия: код и текст.
 */
export interface Reason37 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Исполнитель сообщил об отклонении — сообщение о подозрении на дефект или отклонении с терминала (FR-137).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperatorDeviationReportedV1".
 */
export interface OperatorDeviationReportedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
workplace_id?: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
operator_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
zone_id?: string
/**
 * Что заметил.
 */
description: string
}
/**
 * Исполнитель запросил контроль — запрос контроля с терминала (FR-137).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperatorInspectionRequestedV1".
 */
export interface OperatorInspectionRequestedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
workplace_id?: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
operator_id: string
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key: string
}
/**
 * Исполнитель сменил режим оборудования — ручная смена режима (ручной режим, коррекция подачи или оборотов) — обстоятельство операции, не вина; критическое действие (FR-147, AD-28).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperatorModeChangedV1".
 */
export interface OperatorModeChangedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
workplace_id?: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
operator_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id: string
/**
 * Что изменено (например, «подача 130 %»).
 */
change: string
/**
 * Новый режим.
 */
new_mode?: ("automatic" | "manual" | "override_feed" | "override_speed" | "other")
}
/**
 * Ручное вмешательство в автоматику — исполнитель обошёл блокировку или автоматику (в том числе работа при отказе в допуске к рабочему месту); критическое действие (AD-28).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperatorOverridePerformedV1".
 */
export interface OperatorOverridePerformedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
workplace_id?: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
operator_id: string
/**
 * Что обошли.
 */
bypassed: ("interlock" | "automation" | "workplace_admission" | "other")
reason?: Reason38
}
/**
 * Причина действия: код и текст.
 */
export interface Reason38 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Шаг подтверждён исполнителем — рутинная отметка исполнителя (уровень подписи 1, PIN на смену).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OperatorStepConfirmedV1".
 */
export interface OperatorStepConfirmedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
operation_run_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
workplace_id?: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
operator_id: string
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key: string
/**
 * Шаг ТП внутри операции.
 */
tp_step?: string
}
/**
 * Соединение с внешней системой проверено по кнопке администратора — та же сверка ответной стороны, что при старте адаптера: метаданные, версия контракта (AD-18, AD-47).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OpsIntegrationCheckedV1".
 */
export interface OpsIntegrationCheckedV1 {
/**
 * Внешняя система.
 */
system: ("onec" | "galaktika" | "mes" | "kompas" | "skud" | "ca" | "visionqc" | "operatorvision" | "partner")
/**
 * ok — ответная сторона отвечает, контракт совпал; degraded — контракт не совпал; unreachable — не отвечает; not_supported — у адаптера нет проверки.
 */
result: ("ok" | "degraded" | "unreachable" | "not_supported")
/**
 * Состояние интеграции на момент проверки.
 */
mode?: ("enabled" | "disabled" | "stand")
/**
 * Адрес ответной стороны без секретов.
 */
endpoint?: string
/**
 * Что ответила или почему не ответила ответная сторона.
 */
detail?: string
}
/**
 * Интеграция в режиме degraded — при старте адаптер сверил метаданные и версию контракта внешней системы; расхождение — канал `degraded`, не отправляет (AD-18).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OpsIntegrationDegradedV1".
 */
export interface OpsIntegrationDegradedV1 {
/**
 * Система.
 */
system: ("onec" | "galaktika" | "mes" | "kompas" | "skud" | "ca" | "partner")
/**
 * Состояние канала.
 */
state: ("ok" | "degraded")
/**
 * Что не сошлось.
 */
detail?: string
}
/**
 * Состояние интеграции задано — администратор включил, выключил или переключил «стенд ↔ реальная система» установленную конфигурацией внешнюю систему; процессы подхватывают состояние без перезапуска (AD-47, FR-157; единолично — Д-71).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OpsIntegrationStateSetV1".
 */
export interface OpsIntegrationStateSetV1 {
/**
 * Внешняя система.
 */
system: ("onec" | "galaktika" | "mes" | "kompas" | "skud" | "ca" | "visionqc" | "operatorvision" | "partner")
/**
 * enabled — включена, реальная система; disabled — выключена: исходящие копятся в очереди, входящие отвергаются приёмом; stand — включена, обмен со стендом (эмулятором). В профиле prod stand запрещён.
 */
state: ("enabled" | "disabled" | "stand")
/**
 * Состояние до решения (по умолчанию профиля, если решений не было).
 */
previous?: ("enabled" | "disabled" | "stand")
reason: Reason39
}
/**
 * Причина действия: код и текст.
 */
export interface Reason39 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Сбой обработки изделия — ошибка свёртки или проекции на записи: изделие «обработка остановлена», партиция продолжает (AD-45).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OpsProcessingFailedV1".
 */
export interface OpsProcessingFailedV1 {
/**
 * Потребитель.
 */
consumer: string
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 */
failed_seq: number
/**
 * Ошибка.
 */
error: string
}
/**
 * Повтор обработки запрошен — `ant rebuild --item` после исправления (AD-45).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OpsProcessingRetriedV1".
 */
export interface OpsProcessingRetriedV1 {
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
failure_event_id: string
reason?: Reason40
}
/**
 * Причина действия: код и текст.
 */
export interface Reason40 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Источник отключён — отключение источника или адаптера — критическое действие (AD-28).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OpsSourceDisabledV1".
 */
export interface OpsSourceDisabledV1 {
/**
 * Идентификатор источника событий: устройство, шлюз, терминал, партнёр (`partner:‹код›`); в прогоне сценария — `‹run_id›/‹источник›` (AD-38).
 */
source_id: string
reason: Reason41
}
/**
 * Причина действия: код и текст.
 */
export interface Reason41 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Источник включён — включение источника или адаптера.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "OpsSourceEnabledV1".
 */
export interface OpsSourceEnabledV1 {
/**
 * Идентификатор источника событий: устройство, шлюз, терминал, партнёр (`partner:‹код›`); в прогоне сценария — `‹run_id›/‹источник›` (AD-38).
 */
source_id: string
reason: Reason42
}
/**
 * Причина действия: код и текст.
 */
export interface Reason42 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Параметры аудита установлены — принадлежат Аудитору ИБ: перечень критических типов, интервал и предельный разрыв контрольных точек, отпечаток ключа хранителя, подписчики шины безопасности (AD-8, AD-15).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "PolicyAuditParametersSetV1".
 */
export interface PolicyAuditParametersSetV1 {
/**
 * Дополнительно критические типы сверх каталога.
 * 
 * Items: Тип события: семейство.сущность.действие — три сегмента латиницей (AD-40).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "event_type".
 */
critical_types?: string[]
/**
 * Интервал контрольных точек N, секунд.
 */
checkpoint_interval_s: number
/**
 * Предельный разрыв между точками, секунд.
 */
checkpoint_max_gap_s: number
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
keeper_key_fingerprint: string
/**
 * Подписчики шины безопасности.
 * 
 * Items: Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
security_bus_subscribers?: string[]
}
/**
 * Полномочие выдано — особое право поверх роли с рамками (программа, семейство изделий, тяжесть, типы решений) и сроком; делегирование — тоже полномочие (FR-50, FR-78).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "PolicyAuthorityGrantedV1".
 */
export interface PolicyAuthorityGrantedV1 {
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
person_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
authority_id: string
/**
 * Область действия: путь `здание/цех/участок/рабочее место` (иерархия областей, AD-15).
 */
scope: string
limits?: AuthorityLimits
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
delegated_by?: string
/**
 * Приказ.
 */
order_ref?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_from: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_until?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id?: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
second_signature_by?: string
}
/**
 * Рамки полномочия.
 */
export interface AuthorityLimits {
/**
 * Программы.
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
programs?: string[]
/**
 * Семейства изделий.
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
item_type_ids?: string[]
/**
 * Класс тяжести дефекта по ГОСТ 15467: критический / значительный / малозначительный / неизвестен.
 */
max_severity?: ("critical" | "major" | "minor" | "unknown")
/**
 * Типы решений.
 * 
 * Items: Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
decision_types?: string[]
}
/**
 * Полномочие отозвано — полномочие не действует с указанного момента.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "PolicyAuthorityRevokedV1".
 */
export interface PolicyAuthorityRevokedV1 {
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
person_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
authority_id: string
/**
 * Область действия: путь `здание/цех/участок/рабочее место` (иерархия областей, AD-15).
 */
scope: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
effective_from: string
reason?: Reason43
}
/**
 * Причина действия: код и текст.
 */
export interface Reason43 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Роль назначена в области — человеку назначена роль в области действия («исполнитель — пост сварки 2»); привилегированные выдачи — со второй подписью независимой стороны (AD-11, PRD §11.16).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "PolicyRoleAssignedV1".
 */
export interface PolicyRoleAssignedV1 {
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
person_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
role_id: string
/**
 * Область действия: путь `здание/цех/участок/рабочее место` (иерархия областей, AD-15).
 */
scope: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_from: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_until?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id?: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
second_signature_by?: string
}
/**
 * Роль определена — новая роль — запись политики без изменения кода; роли могут наследовать права других ролей (FR-78, AD-15).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "PolicyRoleDefinedV1".
 */
export interface PolicyRoleDefinedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
role_id: string
/**
 * Название роли по-русски.
 */
title: string
/**
 * Наследуемые роли.
 * 
 * Items: Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
inherits?: string[]
/**
 * Разрешённые действия.
 * 
 * Items: Действие каталога операций `‹модуль›.‹объект›.‹действие›`; `*` — любой сегмент.
 */
actions: string[]
}
/**
 * Роль снята — роль в области больше не действует (FR-78).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "PolicyRoleUnassignedV1".
 */
export interface PolicyRoleUnassignedV1 {
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
person_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
role_id: string
/**
 * Область действия: путь `здание/цех/участок/рабочее место` (иерархия областей, AD-15).
 */
scope: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
effective_from: string
reason?: Reason44
}
/**
 * Причина действия: код и текст.
 */
export interface Reason44 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Правило разделения обязанностей — например, «кто участвовал в изготовлении, не принимает изделие на точке предъявления» (FR-56); «заверитель ≠ подписант» (AD-43).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "PolicySodRuleSetV1".
 */
export interface PolicySodRuleSetV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
rule_id: string
/**
 * Формулировка.
 */
statement: string
/**
 * Несовместимые действия одного лица над одним объектом.
 * 
 * @minItems 2
 * 
 * Items: Действие каталога операций.
 */
conflicting_actions: [string, string, ...(string)[]]
/**
 * Действует.
 */
enabled: boolean
}
/**
 * Цифровое клеймо выдано — цифровое клеймо контролёра: по приказу, одно на вид контроля, с областью и сроком (FR-145); выдачу согласует начальник ОТК (AD-11).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "PolicyStampIssuedV1".
 */
export interface PolicyStampIssuedV1 {
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
person_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
stamp_id: string
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 */
inspection_kind: string
/**
 * Область действия: путь `здание/цех/участок/рабочее место` (иерархия областей, AD-15).
 */
scope: string
/**
 * Приказ.
 */
order_ref: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_from: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_until?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
document_id?: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
second_signature_by?: string
}
/**
 * Цифровое клеймо отозвано — клеймо отозвано (утеря, увольнение, приказ); действия по виду контроля без действующего клейма отклоняются (FR-145).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "PolicyStampRevokedV1".
 */
export interface PolicyStampRevokedV1 {
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
person_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
stamp_id: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
effective_from: string
reason: Reason45
}
/**
 * Причина действия: код и текст.
 */
export interface Reason45 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Физический дефект установлен — новый дефект по ключу (изделие, зона, место) без вида; уточнение вида в новом наблюдении не создаёт новый дефект (FR-37).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "QualityDefectIdentifiedV1".
 */
export interface QualityDefectIdentifiedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
defect_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
zone_id: string
/**
 * Место в зоне.
 */
location?: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
first_observation_event_id: string
/**
 * Текущий вид по последнему наблюдению.
 */
defect_type_code?: string
}
/**
 * Пропуск брака зафиксирован — дефект найден позже, чем мог; связаны ранние наблюдения «не обнаружено» той же зоны и их версии, список изделий той же версии модели — на перепроверку (FR-100).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "QualityEscapeRecordedV1".
 */
export interface QualityEscapeRecordedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
defect_id: string
/**
 * Ранние наблюдения «признаков нет» этой зоны.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
missed_observation_event_ids: string[]
/**
 * Метод пропустившей точки способен выявить этот вид (иначе это не ошибка модели).
 */
method_covers_defect: boolean
/**
 * Версия анализатора, пропустившая дефект.
 */
analyzer_version?: string
}
/**
 * Уверенный результат пропущен к следующему контролю — анализатор сам пропускает к следующему контролю только уверенные «признаков нет» — по делегированному правилу режима 2 и только при уровне доверия паспорта 4; «брак» и «оценка невозможна» всегда идут к человеку; часть пропущенных выборочно перепроверяет человек (FR-48, AD-27, AD-29). Закрывающую точку не проходит.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "QualityInspectionAutoPassedV1".
 */
export interface QualityInspectionAutoPassedV1 {
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
observation_event_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
passport_id: string
/**
 * Уровень доверия паспорта — только 4.
 */
trust_level: number
/**
 * Попало в выборочную перепроверку человеком.
 */
sampled_for_human_recheck: boolean
}
/**
 * Нет данных контроля — ожидаемый по процессу результат не поступил: «неизвестно», не «годно»; задача на контроль (FR-35).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "QualityInspectionMissingV1".
 */
export interface QualityInspectionMissingV1 {
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key: string
/**
 * Точка контроля.
 */
inspection_point?: string
/**
 * Метод контроля (FR-14, FR-27): камера, КИМ, рентген, УЗК, капиллярный, течеискатель, момент затяжки, визуальный человеком, документы поставщика, лаборатория, иное.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "inspection_method".
 */
method?: ("camera" | "cmm" | "radiography" | "ultrasonic" | "penetrant" | "leak_test" | "torque" | "visual_human" | "supplier_documents" | "laboratory" | "other")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
after_operation_run_id?: string
/**
 * Почему нет данных.
 */
missing_reason: ("result_not_received" | "check_skipped" | "point_manual_mode" | "not_covered_by_method" | "unknown")
}
/**
 * Наблюдение связано с известным дефектом — тот же физический дефект увидели ещё раз (другая камера, следующая КТ); число дефектов не растёт (FR-37).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "QualityObservationLinkedV1".
 */
export interface QualityObservationLinkedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
defect_id: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
observation_event_id: string
/**
 * Уточнённый вид, если изменился.
 */
defect_type_code?: string
}
/**
 * Сигнал о признаке дефекта или нарушении — система сверила факт с нормой и картой реакций: это не брак, а первый из четырёх статусов (кейс §2.3); без требования КД — вопрос технологу (FR-48).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "QualitySignalRaisedV1".
 */
export interface QualitySignalRaisedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
signal_id: string
/**
 * На чём основан сигнал.
 */
basis_kind: ("inspection_result" | "equipment_deviation" | "check_skipped" | "damage_on_receipt" | "leak" | "special_process_violation" | "operator_report")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
defect_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
zone_id?: string
/**
 * Вид дефекта по классификатору; неизвестный — с флагом.
 */
defect_type_code?: string
/**
 * Вид есть в классификаторе (FR-125).
 */
defect_type_known?: boolean
/**
 * Класс тяжести дефекта по ГОСТ 15467: критический / значительный / малозначительный / неизвестен.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "severity".
 */
severity: ("critical" | "major" | "minor" | "unknown")
/**
 * Требование КД: характеристика, допуск, ревизия; отсутствует — вопрос инженерам, а не брак.
 */
requirement_ref?: string
/**
 * Доля в базисных пунктах: 0…10000 (10000 = 1,0). Float в контрактах запрещён (AD-4).
 */
analyzer_confidence_bp?: number
/**
 * Доля в базисных пунктах: 0…10000 (10000 = 1,0). Float в контрактах запрещён (AD-4).
 */
observation_quality_bp?: number
/**
 * Реакция карты реакций.
 */
reaction_outcome: ("pass_to_next" | "manual_review" | "isolate" | "question_to_technologist")
/**
 * Карта реакций и строка: `‹id›@‹версия›#‹строка›`.
 */
reaction_map_ref: string
/**
 * Уровень доверия паспорта анализатора, разрешивший действие (AD-29), если сигнал от анализатора.
 */
trust_level?: number
}
/**
 * Производственный календарь — рабочие и нерабочие дни — для сроков в рабочих днях (FR-55, AD-4).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ReferenceCalendarDefinedV1".
 */
export interface ReferenceCalendarDefinedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
calendar_id: string
/**
 * Год.
 */
year: number
/**
 * Еженедельные выходные.
 * 
 * Items: День недели.
 */
weekly_days_off?: ("mon" | "tue" | "wed" | "thu" | "fri" | "sat" | "sun")[]
/**
 * Нерабочие дни сверх еженедельных (праздники).
 * 
 * Items: Календарная дата YYYY-MM-DD (без format: date — диалект AD-20).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "date".
 */
non_working_days: string[]
/**
 * Рабочие дни, выпадающие на еженедельные выходные (перенос выходного, ТК РФ ст. 112).
 * 
 * Items: Календарная дата YYYY-MM-DD (без format: date — диалект AD-20).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "date".
 */
working_days?: string[]
/**
 * Сокращённые дни.
 * 
 * Items: Календарная дата YYYY-MM-DD (без format: date — диалект AD-20).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "date".
 */
shortened_days?: string[]
}
/**
 * Изменение справочника затрагивает изделие — изменение с датой действия в прошлом доходит до изделия адресованной записью межизделийной стадии (AD-31, AD-42).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ReferenceChangeAffectsItemV1".
 */
export interface ReferenceChangeAffectsItemV1 {
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
reference_event_id: string
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 */
reference_kind: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_from: string
}
/**
 * Оборудование определено — станок, источник, средство контроля: вид, место, класс; реестр средств измерений (ГОСТ Р 58876 п. 7.1.5.2).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ReferenceEquipmentDefinedV1".
 */
export interface ReferenceEquipmentDefinedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id: string
/**
 * Вид.
 */
kind: ("cnc_machine" | "welding_source" | "camera" | "cmm" | "leak_tester" | "torque_wrench" | "xray" | "test_bench" | "other")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
location_id: string
/**
 * Название.
 */
name: string
/**
 * Средство измерений — нужна поверка или калибровка.
 */
is_measuring_instrument: boolean
/**
 * Идентификатор источника событий: устройство, шлюз, терминал, партнёр (`partner:‹код›`); в прогоне сценария — `‹run_id›/‹источник›` (AD-38).
 */
source_id?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_from: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_until?: string
}
/**
 * Поверка или калибровка записана — статус поверки на дату — предусловие операции; непригодность — пересмотр прежних результатов (FR-17, ГОСТ Р 58876 п. 7.1.5.2).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ReferenceEquipmentVerifiedV1".
 */
export interface ReferenceEquipmentVerifiedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
equipment_id: string
/**
 * Вид.
 */
kind: ("verification" | "calibration")
/**
 * Итог.
 */
result: ("valid" | "invalid")
/**
 * Календарная дата YYYY-MM-DD (без format: date — диалект AD-20).
 */
valid_until?: string
/**
 * Свидетельство.
 */
certificate_ref?: string
}
/**
 * Соответствие внешнего идентификатора — внешний ID не становится нашим ключом — хранится соответствие; конфликт — сигнал, не перезапись (FR-95, AD-18).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ReferenceExternalIdMappedV1".
 */
export interface ReferenceExternalIdMappedV1 {
/**
 * Внешняя система.
 */
system: ("onec" | "galaktika" | "mes" | "kompas" | "skud" | "partner" | "other")
/**
 * Вид объекта.
 */
object_kind: ("item_type" | "order" | "lot" | "supplier" | "warehouse" | "item" | "person" | "equipment" | "contract")
/**
 * ID во внешней системе.
 */
external_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
internal_id: string
}
/**
 * Тип изделия определён — номенклатура: обозначение, ревизия, состав, зоны изделия, способ маркировки (FR-46, AD-31).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ReferenceItemTypeDefinedV1".
 */
export interface ReferenceItemTypeDefinedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
item_type_id: string
/**
 * Обозначение по КД.
 */
designation: string
/**
 * Наименование.
 */
name: string
/**
 * Ревизия.
 */
revision: string
/**
 * Зоны.
 */
zones?: ZoneDef[]
/**
 * Состав.
 */
components?: ComponentDef[]
/**
 * Тип носителя идентификатора (AD-16): DPM DataMatrix / бирка с QR / тара и ячейка / сопроводительная карта / контекст поста / ручной ввод / внутренний ID системы.
 */
marking?: ("dpm_datamatrix" | "tag_qr" | "container_cell" | "route_card" | "post_context" | "manual_entry" | "internal_id")
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_from: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_until?: string
}
/**
 * Зона изделия.
 */
export interface ZoneDef {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
zone_id: string
/**
 * Название.
 */
name: string
/**
 * Вид зоны.
 */
kind: ("weld_section" | "joint" | "hole" | "surface" | "cavity" | "groove" | "other")
}
/**
 * Позиция состава.
 */
export interface ComponentDef {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
item_type_id: string
/**
 * Количество.
 */
quantity: number
/**
 * Учитывается партией.
 */
lot_tracked: boolean
}
/**
 * Место определено — иерархия областей здание → цех → участок → рабочее место, линии, склады и изоляторы; цех — дорожка BPMN со своим складом (FR-130, AD-15).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ReferenceLocationDefinedV1".
 */
export interface ReferenceLocationDefinedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
location_id: string
/**
 * Вид места.
 */
kind: ("building" | "workshop" | "line" | "station" | "workplace" | "warehouse" | "isolator" | "storage")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
parent_id?: string
/**
 * Путь области для политики доступа.
 */
scope: string
/**
 * Название.
 */
name: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
warehouse_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
access_zone_id?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_from: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
valid_until?: string
}
/**
 * График смен — смены по участкам: начало, конец; смена-шаблон повторяется каждый (рабочий) день до repeat_until (FR-81).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "ReferenceShiftScheduledV1".
 */
export interface ReferenceShiftScheduledV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
shift_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
location_id: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
starts_at: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
ends_at: string
/**
 * Название смены.
 */
name?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
repeat_until?: string
/**
 * Повторять только в рабочие дни производственного календаря.
 */
working_days_only?: boolean
}
/**
 * Отказ в доступе — вызов API или действие отклонено по политике, месту сеанса или доменному гарду (FR-85, AD-24).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SecurityAccessDeniedV1".
 */
export interface SecurityAccessDeniedV1 {
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
person_id?: string
/**
 * Действие каталога операций.
 */
action_id: string
/**
 * Поток записи (AD-39): `item:‹item_id›`, `‹вид_объекта›:‹id›` или `global`.
 */
object_ref?: string
/**
 * Код ошибки из contracts/errors.yaml.
 */
problem_code: string
}
/**
 * Отказ в допуске к рабочему месту — условие допуска не выполнено; отказ фиксируется в журнале критических действий (FR-77, FR-83).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SecurityAdmissionDeniedV1".
 */
export interface SecurityAdmissionDeniedV1 {
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
person_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
workplace_id: string
/**
 * Невыполненные условия.
 * 
 * @minItems 1
 * 
 * Items: Невыполненное условие.
 */
failed_checks: [("not_in_zone" | "no_role_in_scope" | "qualification_invalid" | "not_assigned" | "no_token" | "wrong_pin"), ...(("not_in_zone" | "no_role_in_scope" | "qualification_invalid" | "not_assigned" | "no_token" | "wrong_pin"))[]]
}
/**
 * Неудачный вход — ошибка аутентификации (логин и пароль) — событие шины безопасности (AD-15, AD-24).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SecurityAuthFailedV1".
 */
export interface SecurityAuthFailedV1 {
/**
 * Логин попытки.
 */
login?: string
/**
 * Адрес клиента.
 */
client_ip?: string
/**
 * Причина.
 */
reason: ("bad_credentials" | "account_locked" | "account_pending" | "rate_limited")
}
/**
 * Критическое действие — запись цепочки критических действий: действие, объект, было → стало, кто, полномочие и клеймо с ревизией политики, основание, материалы, ссылка на подписанную запись основного журнала; формирует только `domain/security.BuildCA` в той же транзакции (AD-28). Отмена — новой записью с `cancels`.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SecurityCriticalActionRecordedV1".
 */
export interface SecurityCriticalActionRecordedV1 {
/**
 * Номер `CA-‹n›` — позиция в цепочке критических действий (в хеш состояния не входит).
 */
ca_no: number
/**
 * Группа (AD-28).
 */
ca_group: ("product_decision" | "nc_decision" | "cause" | "risk_scope" | "control_change" | "authority" | "protected_data" | "admin_security")
/**
 * Тип события: семейство.сущность.действие — три сегмента латиницей (AD-40).
 */
action_type: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
main_event_id: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
main_commit: string
/**
 * Поток записи (AD-39): `item:‹item_id›`, `‹вид_объекта›:‹id›` или `global`.
 */
object_ref: string
/**
 * Было (кратко).
 */
before: string
/**
 * Стало (кратко).
 */
after: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
actor_id?: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
attested_by?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
authority_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
stamp_id?: string
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 */
policy_seq?: number
/**
 * Основание и материалы.
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
basis_event_ids: string[]
/**
 * Отменяемое критическое действие `CA-‹n›`.
 */
cancels?: string
cancel_reason?: Reason46
}
/**
 * Причина действия: код и текст.
 */
export interface Reason46 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Конфликт идемпотентности — тот же `source_id + event_id`, другое содержимое — конфликт целостности; частота ограничена по источнику (FR-31, AD-7).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SecurityIdempotencyConflictV1".
 */
export interface SecurityIdempotencyConflictV1 {
/**
 * Идентификатор источника событий: устройство, шлюз, терминал, партнёр (`partner:‹код›`); в прогоне сценария — `‹run_id›/‹источник›` (AD-38).
 */
source_id: string
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
event_id: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
first_payload_digest: string
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
conflicting_payload_digest: string
}
/**
 * Отчёт верификатора получен — ant забрал последний подписанный отчёт верификатора у хранителя; индикатор «по данным сервера» (AD-46).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SecurityIntegrityCheckedV1".
 */
export interface SecurityIntegrityCheckedV1 {
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
report_digest: string
/**
 * Общий вердикт.
 */
verdict: ("intact" | "intact_with_reservations" | "violated")
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 */
checked_up_to_seq: number
/**
 * Отпечаток по хешу формата цепочки с префиксом алгоритма: `streebog256:‹64 hex›` (AD-44). Им же адресуются материалы.
 */
verifier_build?: string
}
/**
 * Нарушение целостности обнаружено — звено цепочки не сходится, расхождение с контрольной точкой, проекция расходится с журналом (CA-…) — индикатор целостности на столах (FR-74).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SecurityIntegrityViolatedV1".
 */
export interface SecurityIntegrityViolatedV1 {
/**
 * Вид нарушения.
 */
violation: ("chain_link_mismatch" | "checkpoint_mismatch" | "projection_mismatch" | "source_seq_gap" | "late_write" | "signature_invalid")
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 */
seq?: number
/**
 * Номер критического действия `CA-‹n›`, если затронуто.
 */
ca_ref?: string
/**
 * Что и где.
 */
detail: string
}
/**
 * Тревога хранителя — головы цепочек не приходили дольше 2N секунд, попытка вилки или отката, разрыв контрольных точек (AD-8).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SecurityKeeperAlertV1".
 */
export interface SecurityKeeperAlertV1 {
/**
 * Вид тревоги.
 */
alert: ("heads_silent" | "fork_attempt" | "rollback_attempt" | "checkpoint_gap")
/**
 * Цепочка.
 */
chain: ("main" | "ca")
/**
 * Позиция записи в основной цепочке журнала (порядок знания, AD-37).
 */
last_seq?: number
/**
 * Подробности.
 */
detail?: string
}
/**
 * Тревога по ключу — повторный ключ человека, чужой ключ (пользователь сеанса ≠ субъект ключа), расхождение локального журнала агента с сервером (AD-11, AD-14).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SecurityKeyAlertV1".
 */
export interface SecurityKeyAlertV1 {
/**
 * Ссылка на ключ подписанта: `key_id@версия` (AD-10).
 */
key_ref?: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
person_id?: string
/**
 * Вид тревоги.
 */
alert: ("duplicate_key" | "foreign_key" | "agent_journal_mismatch" | "level1_rate_exceeded")
/**
 * Подробности.
 */
detail?: string
}
/**
 * Отклонение присутствия — «ключ вставлен, владельца нет в зоне» — тревога администратору; «по графику должен быть, ключа нет» — уведомление мастеру (FR-84).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SecurityPresenceDeviationV1".
 */
export interface SecurityPresenceDeviationV1 {
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
person_id: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
workplace_id: string
/**
 * Вид.
 */
deviation: ("token_without_presence" | "scheduled_without_token")
}
/**
 * Недействительная подпись — подпись сообщения или документа не прошла проверку: неизвестный, отозванный, чужой ключ, понижение профиля, изменённый пакет (FR-26, FR-68, AD-10).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SecuritySignatureInvalidV1".
 */
export interface SecuritySignatureInvalidV1 {
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
subject_event_id?: string
/**
 * Идентификатор источника событий: устройство, шлюз, терминал, партнёр (`partner:‹код›`); в прогоне сценария — `‹run_id›/‹источник›` (AD-38).
 */
source_id?: string
/**
 * Ссылка на ключ подписанта: `key_id@версия` (AD-10).
 */
key_ref?: string
/**
 * Что не так; `unsigned` — подписи нет там, где политика её требует.
 */
failure: ("unknown_key" | "revoked_key" | "foreign_key" | "profile_downgrade" | "payload_tampered" | "bad_signature" | "key_unavailable" | "class_not_allowed" | "unsigned")
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
quarantine_event_id?: string
}
/**
 * Инъекция цифрового стенда — кнопка поверх идущего прогона: повтор, опоздавшее событие, испорченный кадр, сбой станка, потеря данных (FR-152).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SimulationInjectionAppliedV1".
 */
export interface SimulationInjectionAppliedV1 {
/**
 * Идентификатор прогона сценария (AD-38); в профиле prod отсутствует.
 */
run_id: string
/**
 * Кнопка.
 */
injection: ("duplicate_event" | "late_event" | "corrupt_frame" | "machine_fault" | "data_loss" | "tamper_outside" | "light_change")
/**
 * Идентификатор UUID в нижнем регистре (RFC 9562).
 */
target_event_id?: string
/**
 * Номер нажатия кнопки в прогоне (1, 2, …): повторное нажатие — новая запись, не повтор (AD-7).
 */
press?: number
/**
 * Записи, внесённые кнопкой через обычный приём (повтор — ни одной).
 * 
 * @maxItems 100
 * 
 * Items: Идентификатор UUID в нижнем регистре (RFC 9562).
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "uuid".
 */
event_ids?: string[]
}
/**
 * Прогон завершён — итог прогона; автосверка — табло «ожидалось → получилось» (FR-108).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SimulationRunFinishedV1".
 */
export interface SimulationRunFinishedV1 {
/**
 * Идентификатор прогона сценария (AD-38); в профиле prod отсутствует.
 */
run_id: string
/**
 * Итог.
 */
outcome: ("completed" | "stopped" | "failed")
}
/**
 * Прогон на паузе — пауза останавливает поток и часы; столы показывают состояние на момент паузы (FR-129).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SimulationRunPausedV1".
 */
export interface SimulationRunPausedV1 {
/**
 * Идентификатор прогона сценария (AD-38); в профиле prod отсутствует.
 */
run_id: string
/**
 * Почему.
 */
reason: ("user" | "waiting_for_decision")
}
/**
 * Прогон продолжен — поток возобновлён без потерь и дублей (FR-129).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SimulationRunResumedV1".
 */
export interface SimulationRunResumedV1 {
/**
 * Идентификатор прогона сценария (AD-38); в профиле prod отсутствует.
 */
run_id: string
}
/**
 * Прогон сценария запущен — `run_id`, seed, версия сценария, скорость; всё, что прогон меняет, — записями с `run_id` (AD-38).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "SimulationRunStartedV1".
 */
export interface SimulationRunStartedV1 {
/**
 * Идентификатор прогона сценария (AD-38); в профиле prod отсутствует.
 */
run_id: string
/**
 * Сценарий.
 */
scenario_id: string
/**
 * Версия сценария.
 */
scenario_version: string
/**
 * Seed генератора.
 */
seed: number
/**
 * Скорость ×1…×1000.
 */
speed: number
/**
 * Режим.
 */
mode: ("interactive" | "autocheck" | "load")
}
/**
 * Уведомление — адресное: информация / тревога / запрос решения; только тем, у кого есть действие (FR-57).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "TaskNotificationSentV1".
 */
export interface TaskNotificationSentV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
notification_id: string
/**
 * Тип.
 */
severity: ("info" | "alarm" | "decision_request")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
recipient_role_id?: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
recipient_person_id?: string
/**
 * Поток записи (AD-39): `item:‹item_id›`, `‹вид_объекта›:‹id›` или `global`.
 */
subject_ref: string
/**
 * Ключ текста интерфейса (ui-texts).
 */
text_key: string
/**
 * Параметры текста.
 */
params?: {
[k: string]: string | undefined
}
}
/**
 * Уведомление отозвано — основание уведомления исчезло при пересвёртке (AD-3).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "TaskNotificationWithdrawnV1".
 */
export interface TaskNotificationWithdrawnV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
notification_id: string
}
/**
 * Задача подтверждена — исполнитель подтвердил выполнение или принятие задачи.
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "TaskTaskAcknowledgedV1".
 */
export interface TaskTaskAcknowledgedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
task_id: string
/**
 * Итог.
 */
outcome: ("done" | "accepted" | "declined")
/**
 * Комментарий.
 */
note?: string
}
/**
 * Задача поставлена — адресная задача тому, у кого есть действие: владелец, срок, подтверждение (FR-57); физические задачи, доп. проверка, переподписание, «основание защиты изменилось — пересмотрите» (AD-3).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "TaskTaskCreatedV1".
 */
export interface TaskTaskCreatedV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
task_id: string
/**
 * Вид задачи.
 */
kind: ("physical_move" | "isolate_move" | "recheck" | "decision_required" | "review_after_new_data" | "protection_basis_changed" | "resign" | "remark_carrier" | "remove_temporary_carrier" | "inspection_missing" | "admin_resend" | "process_step" | "other")
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
assignee_role_id: string
/**
 * Условный идентификатор (псевдоним) сотрудника; соответствие человеку хранит модуль access.
 */
assignee_person_id?: string
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
location_id?: string
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
due_at?: string
/**
 * Поток записи (AD-39): `item:‹item_id›`, `‹вид_объекта›:‹id›` или `global`.
 */
subject_ref: string
/**
 * Что сделать.
 */
title: string
/**
 * Внутренний идентификатор изделия: `код_предприятия:локальный_id` (AD-16). Из метки не выводится; в прогоне локальная часть несёт префикс прогона.
 */
item_id?: string
/**
 * Метка изделия для людей: номер с бирки (Ф-001), DM-код или номер из id.
 */
item_label?: string
/**
 * Задача процесса (kind process_step): операция API, которой исполнитель продвигает изделие (process.movement.receive, process.operation.start и т. п.).
 */
operation_id?: string
/**
 * Стабильный ключ шага процесса из расширения BPMN (`ant:properties/@stepKey`, AD-17).
 */
step_key?: string
}
/**
 * Задача снята — при пересвёртке реакция исчезла — задача снимается (AD-3).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "TaskTaskWithdrawnV1".
 */
export interface TaskTaskWithdrawnV1 {
/**
 * Идентификатор объекта системы или справочника: ASCII, без пробелов.
 */
task_id: string
reason?: Reason47
}
/**
 * Причина действия: код и текст.
 */
export interface Reason47 {
/**
 * Машинный код: латиница в нижнем регистре, цифры, подчёркивание.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "code".
 */
code?: string
/**
 * Текст на русском для человека.
 * 
 * This interface was referenced by `AntDefsV1`'s JSON-Schema
 * via the `definition` "text".
 */
text: string
}
/**
 * Режим часов установлен — свойство журнала: `system` или `scenario`; фиксируется генезисом, меняется только сбросом окружения (AD-37).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "TimeClockModeSetV1".
 */
export interface TimeClockModeSetV1 {
/**
 * Режим часов.
 */
mode: ("system" | "scenario")
}
/**
 * Тик виртуальных часов сценария — доменное «сейчас» в режиме `scenario` — последняя такая запись; ведёт simulation (AD-37).
 * 
 * This interface was referenced by `EventsContracts`'s JSON-Schema
 * via the `definition` "TimeClockTickedV1".
 */
export interface TimeClockTickedV1 {
/**
 * Момент времени: RFC 3339 UTC, ровно три знака после секунд (YYYY-MM-DDTHH:MM:SS.mmmZ).
 */
now: string
/**
 * Скорость ×1…×1000.
 */
speed: number
}

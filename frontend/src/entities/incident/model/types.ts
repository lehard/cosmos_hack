/**
 * Входные данные экранов разбора (эпик 12): разбор обстоятельств, общие факторы,
 * гипотезы, похожие случаи, область риска, группы несоответствий.
 *
 * Поля названы как в контракте семейства `incident` (contracts/events/incident)
 * и проекции `analysis.circumstances` (AD-29): когда операции чтения появятся
 * в contracts/openapi.yaml, сгенерированные типы подставляются сюда без
 * переименований. Смысл полей — только контрактный (NFR-UI-4): уверенность ≠
 * вероятность вины, «под подозрением» ≠ брак, гипотеза ≠ причина.
 */

/** Дорожка экрана «Разбор обстоятельств» (FR-153): изделие, человек, оборудование. */
export type CircumstanceLane = 'item' | 'person' | 'equipment'

/** Все дорожки в порядке показа. */
export const CIRCUMSTANCE_LANES: readonly CircumstanceLane[] = ['item', 'person', 'equipment']

/**
 * Ссылка на запись журнала с тем, что нужно для подписи: тип записи из каталога
 * (`семейство.сущность.действие`), уточнение (исход контроля, вид отклонения,
 * состояние) и параметры подписи, уже приведённые проекцией к тексту.
 */
export interface JournalRecordRef {
  /** `event_id` записи. */
  event_id: string
  /** Тип записи каталога, например `equipment.deviation.detected`. */
  event_type: string
  /**
   * Уточнение внутри типа: `outcome` контроля (`defect_indicated` …),
   * `deviation_kind` отклонения, `cycle_started` / `cycle_finished`, `condition`.
   */
  variant?: string | null
  /** Время возникновения, RFC 3339 UTC (AD-37). */
  occurred_at: string
  /** Параметры подписи: метод, параметр, значение, уставка, шаг… */
  params?: Record<string, string | number>
}

/** Запись на дорожке разбора обстоятельств — строка проекции `analysis.circumstances`. */
export interface CircumstanceRecord extends JournalRecordRef {
  /** Дорожка. */
  lane: CircumstanceLane
  /** Конец интервала (цикл, отклонение), если запись — интервал. */
  ended_at?: string | null
  /** Позиция записи в журнале (`seq`) — для перехода к записи. */
  journal_seq?: number | null
  /** Адреса материалов: кадры, протоколы (`evidence_refs`). */
  evidence_refs?: string[]
  /** Связанные записи — подсвечиваются вместе с выбранной. */
  related_event_ids?: string[]
  /** Вид источника факта (FR-140), например `machine`, `manual`. */
  source_kind?: string | null
}

/** Окно возможного возникновения: от последнего подтверждённо нормального состояния до первой находки (FR-58). */
export interface CausalWindow {
  /** `causal_window_start`. */
  start: string
  /** `causal_window_end` — первая находка. */
  end: string
  /** Запись последнего подтверждённо нормального состояния. */
  lower_bound_event_id?: string | null
  /** Запись первой находки. */
  upper_bound_event_id?: string | null
}

/** Выполнение операции, вокруг которой идёт разбор (профиль выполнения, FR-148). */
export interface OperationSpan {
  /** `operation_run_id`. */
  operation_run_id: string
  /** Название операции по описанию процесса. */
  label: string
  /** `operation_started_at`. */
  started_at: string
  /** `operation_finished_at`; null — не завершена. */
  finished_at: string | null
}

/** Нехватка сведений — перечисление `missing_information` из `incident.hypothesis.computed`. */
export type MissingInformation =
  | 'tool_unknown'
  | 'cycle_end_time_unknown'
  | 'no_observation_after_operation'
  | 'no_observation_before_operation'
  | 'operator_unknown'
  | 'equipment_log_missing'
  | 'other'

/** Данные экрана «Разбор обстоятельств» по одному несоответствию (FR-58, FR-153). */
export interface CircumstancesModel {
  /** Несоответствие. */
  nc_id: string
  /** Выполнение операции; null — не установлено. */
  operation: OperationSpan | null
  /** Окно возможного возникновения; null — определить нельзя. */
  window: CausalWindow | null
  /** Записи трёх дорожек. */
  records: CircumstanceRecord[]
  /** Нехватка сведений. */
  missing_information: MissingInformation[]
  /** Категоричный ли вывод; при недостатке сведений — false. */
  conclusion_is_categorical: boolean
}

/** Вид общего фактора (FR-135, FR-61). */
export type FactorKind = 'machine' | 'tool' | 'fixture' | 'program' | 'performer' | 'material_batch'

/** Строка таблицы общих факторов «сколько из N». */
export interface CommonFactorRow {
  /** Вид фактора. */
  factor: FactorKind
  /** Самое частое значение фактора в группе; null — неизвестно. */
  value: string | null
  /** У скольких несоответствий группы фактор совпадает с `value`. */
  matches: number
  /** Сколько разных значений фактора в группе. */
  distinct_values: number
}

/** Общие факторы по группе несоответствий. */
export interface CommonFactorsModel {
  /** Ключ группы (вид дефекта × операция × оборудование). */
  group_key: string
  /** Подпись группы. */
  group_label: string
  /** N — несоответствий в группе. */
  nc_count: number
  /** Факторы. */
  rows: CommonFactorRow[]
}

/** Категория гипотезы причины — перечисление контракта (FR-59). */
export type CauseCategory = 'incoming' | 'equipment' | 'performer' | 'handling' | 'assembly' | 'documentation' | 'not_established'

/** Статус гипотезы: предложена системой / записана человеком / подтверждена / отклонена. */
export type HypothesisStatus = 'proposed_by_system' | 'recorded' | 'confirmed' | 'rejected'

/** Гипотеза причины с доводами «за» и «против» (FR-59, FR-135). */
export interface Hypothesis {
  /** Идентификатор гипотезы. */
  hypothesis_id: string
  /** Категория. */
  category: CauseCategory
  /** Ветка: почему возник / почему пропустили. */
  branch?: 'why_made' | 'why_missed' | null
  /** Формулировка (у записанных человеком). */
  statement?: string | null
  /** Статус. */
  status: HypothesisStatus
  /** Уверенность вывода в базисных пунктах — не вероятность вины. */
  confidence_bp?: number | null
  /** Доводы «за» — записи журнала. */
  supporting: JournalRecordRef[]
  /** Доводы «против». */
  contradicting: JournalRecordRef[]
  /** Что измерить, чтобы проверить гипотезу. */
  measurement_hint?: string | null
}

/** Результат меры похожего случая — коды словаря корректирующих действий. */
export type MeasureResult = 'assigned' | 'implemented' | 'effective' | 'failed'

/** Похожий прошлый случай: по виду дефекта, операции и оборудованию, правилами (FR-60). */
export interface SimilarCase {
  /** Несоответствие. */
  nc_id: string
  /** Номер для людей. */
  number: string
  /** Категория причины; null — не установлена. */
  cause_category: CauseCategory | null
  /** Причина подтверждена человеком (иначе — гипотеза). */
  cause_confirmed: boolean
  /** Мера; null — не назначена. */
  measure: string | null
  /** Результат меры; null — неизвестен. */
  result: MeasureResult | null
}

/** Гипотезы по несоответствию — версия вывода `incident.hypothesis.computed` и записи людей. */
export interface HypothesesModel {
  /** Несоответствие. */
  nc_id: string
  /** Версия гипотез. */
  version: number
  /** Гипотезы. */
  hypotheses: Hypothesis[]
  /** Нехватка сведений. */
  missing_information: MissingInformation[]
  /** Категоричный ли вывод. */
  conclusion_is_categorical: boolean
  /** Похожие случаи. */
  similar_cases: SimilarCase[]
}

/** Разбивка области: в производстве / ушли дальше / собраны / отгружены (FR-61). */
export interface ScopeBreakdown {
  in_production: number
  moved_on: number
  assembled: number
  shipped: number
}

/** Места изделий области — ключи разбивки. */
export type ScopeLocation = keyof ScopeBreakdown

/** Порядок мест в разбивке. */
export const SCOPE_LOCATIONS: readonly ScopeLocation[] = ['in_production', 'moved_on', 'assembled', 'shipped']

/** Как появилась версия области: вычислена правилом / расширена / сужена человеком. */
export type ScopeChange = 'computed' | 'expanded' | 'narrowed'

/** Версия области риска — каждая правка новая (FR-61). */
export interface ScopeVersion {
  /** `scope_version`. */
  scope_version: number
  /** Вид правки: `incident.scope.computed` / `expanded` / `narrowed`. */
  change: ScopeChange
  /** Размер области после версии. */
  size: number
  /** Время записи правки. */
  recorded_at: string
  /** Автор (псевдоним); null — правило системы. */
  author: string | null
  /** Имя автора для людей (сервер); нет — показать код автора. */
  author_name?: string | null
  /** Основание: код и текст (`reason`). */
  reason: { code?: string | null; text: string } | null
  /** Доказательства — `event_id` записей (`evidence_event_ids` / `basis`). */
  evidence_event_ids: string[]
  /** Разбивка после версии. */
  breakdown: ScopeBreakdown
}

/** Что известно об изделии в инциденте — ось `incident` словаря статусов (FR-62). */
export type IncidentKnown = 'confirmed' | 'suspect' | 'excluded' | 'unknown'

/** Что делать с изделием в инциденте — словарь `incident_action` (FR-62). */
export type IncidentAction = 'observe' | 'check' | 'block' | 'release'

/** Изделие в области риска: две оси статуса и место. */
export interface ScopeItem {
  /** Изделие. */
  item_id: string
  /** Номер для людей. */
  label: string
  /** Что известно. */
  known: IncidentKnown
  /** Что делать. */
  action: IncidentAction
  /** Где находится. */
  location: ScopeLocation
}

/** Область риска инцидента с версиями (FR-61, FR-62). */
export interface RiskScopeModel {
  /** Инцидент. */
  incident_id: string
  /** Подпись инцидента. */
  incident_label: string
  /** Общий фактор, по которому собрана область. */
  common_factor: { factor: FactorKind; value: string } | null
  /** Окно области. */
  window: { start: string; end: string } | null
  /** Последнее подтверждённо нормальное состояние; null — неизвестно. */
  last_known_good: { label: string; at: string } | null
  /** Версии по возрастанию. */
  versions: ScopeVersion[]
  /** Изделия текущей версии. */
  items: ScopeItem[]
  /** Отгружено другим предприятиям. */
  shipped_to_partners?: number
}

/** Статус системного расследования — словарь `ncInvestigation`. */
export type NcInvestigationStatus =
  | 'not_required'
  | 'not_started'
  | 'in_progress'
  | 'hypothesis_only'
  | 'cause_confirmed'
  | 'cause_not_established'
  | 'measures_assigned'
  | 'effectiveness_check'
  | 'closed'

/** Группа несоответствий: вид дефекта × операция × оборудование (стол технолога, «Разбор причин»). */
export interface NcGroup {
  /** Ключ группы. */
  group_key: string
  /** Вид дефекта. */
  defect_type: string
  /** Операция. */
  operation: string
  /** Оборудование. */
  equipment: string
  /** Несоответствий в группе. */
  nc_count: number
  /** Статус расследования группы. */
  investigation: NcInvestigationStatus
  /** Последняя находка. */
  last_found_at: string
  /** Несоответствия группы (первое открывается в разборе обстоятельств); может быть пусто. */
  nc_ids?: string[]
}

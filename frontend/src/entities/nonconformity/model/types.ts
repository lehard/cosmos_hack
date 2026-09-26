/**
 * Входные данные карточки несоответствия, очереди «Ждут моего решения» и
 * панели решений (эпик 11; FR-51…FR-55, FR-146; кейс §2.3, §4.5).
 *
 * Поля — как в семействах `quality` (сигнал), `inspection` (наблюдение),
 * `decision` (решения людей и сдерживание) и в метаданных реакции конверта
 * (`reaction.version`, `supersedes`, `revised_due_to`, `causes` — версии вывода
 * системы, FR-32). Перечисления берутся из сгенерированных типов событий.
 *
 * Карточка раздельно держит четыре слоя (FR-51): исходный сигнал — неизменен;
 * анализ системы — версиями; решения людей — записями с подписью; итоговый
 * статус — осями словаря статусов. «Сигнал» ≠ «подтверждённое несоответствие»
 * ≠ «предполагаемая причина» ≠ «подтверждённая ошибка».
 */
import type {
  DecisionDispositionSetV1,
  DecisionPresentationResolvedV1,
  InspectionResultRecordedV1,
  QualitySignalRaisedV1,
} from '@/shared/contracts/events'
import type { DrillRef } from '@/shared/model/drill'
import type { ItemStatuses, QualityStatus, RecordSignature, Reliability, SourceKind } from '@/entities/item'
import type { MissingInformation, NcInvestigationStatus, SimilarCase } from '@/entities/incident'

export type { MissingInformation, NcInvestigationStatus, SimilarCase }

/** Статус несоответствия «по изделию» — словарь `statuses.ncByItem` (FR-51). */
export type NcByItemStatus =
  | 'draft'
  | 'isolated'
  | 'awaiting_disposition'
  | 'awaiting_approvals'
  | 'in_execution'
  | 'awaiting_reinspection'
  | 'execution_verified'
  | 'closed'
  | 'overdue'

/** Статус сигнала — словарь `statuses.signal`. */
export type SignalStatus = 'under_review' | 'confirmed' | 'rejected' | 'recheck_assigned'

/** Тяжесть дефекта по ГОСТ 15467 (`severity`). */
export type Severity = QualitySignalRaisedV1['severity']

/** Метод контроля (`inspection_method`). */
export type InspectionMethod = InspectionResultRecordedV1['method']

/** Вариант решения по изделию (`decision.disposition.set.disposition`). */
export type Disposition = DecisionDispositionSetV1['disposition']

/** Решение на точке предъявления. */
export type PresentationResolution = DecisionPresentationResolvedV1['resolution']

/** Реакция карты реакций на сигнал (`quality.signal.raised.reaction_outcome`). */
export type ReactionOutcome = QualitySignalRaisedV1['reaction_outcome']

/**
 * Наблюдение — результат контроля, на котором основан сигнал: данные
 * `inspection.result.recorded` плюс координаты записи.
 */
export type Observation = InspectionResultRecordedV1 & {
  event_id: string
  occurred_at: string
  source_kind?: SourceKind | null
  reliability?: Reliability | null
}

/**
 * Исходный сигнал (FR-51): данные `quality.signal.raised` и наблюдение.
 * Решения людей его не меняют — отклонение сигнала — новая запись (FR-51).
 */
export type SourceSignal = QualitySignalRaisedV1 & {
  event_id: string
  raised_at: string
  /** Вид дефекта для людей — из классификатора (FR-125); null — вида нет в классификаторе. */
  defect_type_label?: string | null
  /** Зона для людей. */
  zone_label?: string | null
  /** Наблюдение; null — сигнал не от контроля (отклонение оборудования, сообщение исполнителя). */
  observation: Observation | null
}

/** Требование КД, с которым сверяется факт; нет требования — вопрос технологу, а не брак (FR-48). */
export interface Requirement {
  /** `requirement_ref` сигнала. */
  ref: string
  characteristic?: string | null
  /** Допуск текстом с единицей, как в КД. */
  tolerance?: string | null
  /** Ревизия КД. */
  design_revision?: string | null
}

/** Запись журнала в карточке: обстоятельство, довод, основание. */
export interface CardRecord {
  event_id: string
  event_type: string
  /** Уточнение внутри типа (исход контроля, вид отклонения…). */
  variant?: string | null
  occurred_at: string
  /** Параметры подписи, приведённые проекцией к тексту. */
  params?: Record<string, string | number>
  source_kind?: SourceKind | null
  reliability?: Reliability | null
}

/** Выполнение операции, на которой возник признак (профиль выполнения, FR-148). */
export interface OperationContext {
  operation_run_id: string
  /** Название операции по описанию процесса. */
  label: string
  started_at: string | null
  finished_at: string | null
  /** Станок; null — неизвестен. */
  equipment: string | null
  /** Инструмент и ресурс «73 из 75». */
  tool: { label: string; used?: number | null; limit?: number | null } | null
  /** Программа и ревизия. */
  program: { ref: string; revision?: string | null } | null
  /** Исполнитель (псевдоним); null — «исполнитель неизвестен», не додумываем. */
  performer: string | null
  /** Повтор операции — ссылка на предыдущее выполнение (FR-47). */
  rework_of?: string | null
}

/** Точка истории зоны: какой контроль и когда. */
export interface ZoneCheck {
  event_id: string
  /** Точка контроля (`KT-3`). */
  checkpoint: string
  at: string
}

/** «Что произошло»: до операции / операция / после и история зоны (PRD §3a). */
export interface NcCircumstances {
  before: CardRecord[]
  operation: OperationContext | null
  after: CardRecord[]
  /** Последняя чистая проверка зоны и первая с признаком. */
  zone_history: { last_clean: ZoneCheck | null; first_defective: ZoneCheck | null }
  /** Состояние оборудования во время операции: отклонения, ручные изменения режима. */
  equipment: CardRecord[]
  /** Действия исполнителя на операции — обстоятельство, а не вина. */
  performer_actions: CardRecord[]
}

/**
 * Версия вывода системы (FR-32, FR-50, AD-3): метаданные реакции конверта и
 * то, что система предлагает. Пересмотр — новая версия; прежняя остаётся.
 */
export interface SystemConclusion {
  /** Версия слота реакции (`reaction.version`): 1 — первая. */
  version: number
  /** `event_id` реакции. */
  event_id: string
  /** Заменяемая версия (`reaction.supersedes`). */
  supersedes: string | null
  /** Когда вычислено (время записи). */
  computed_at: string
  /** Правило (`reaction.rule_id`) и ревизия нормативного слоя (`rule_rev`). */
  rule_id: string
  rule_rev: string | null
  /** Режим автоматизации правила 1–5 (FR-50). */
  automation_mode: number
  /** Реакция карты реакций. */
  reaction_outcome: ReactionOutcome
  /** Строка карты реакций `‹id›@‹версия›#‹строка›`. */
  reaction_map_ref: string
  /** Тяжесть, уверенность и качество наблюдения, по которым сработала строка. */
  severity: Severity
  analyzer_confidence_bp?: number | null
  observation_quality_bp?: number | null
  /** Подсказанная доп. проверка (режим 3 «предлагает — человек утверждает»). */
  suggested_recheck?: { method: InspectionMethod } | null
  /** Основания — записи-причины вывода (`reaction.causes`). */
  basis: CardRecord[]
  /** Альтернативные объяснения — тексты сервера. */
  alternatives: string[]
  /** Чего не хватает для категоричного вывода. */
  missing_information: MissingInformation[]
  /** Запись, из-за которой вывод пересмотрен (`reaction.revised_due_to`); null у первой версии. */
  revised_due_to: CardRecord | null
}

/** Решение человека по сигналу или несоответствию (FR-51, FR-52, FR-53). */
export interface HumanDecision {
  event_id: string
  /** Тип записи `decision.*` из каталога. */
  event_type: string
  /** Уточнение: вариант решения, решение на точке предъявления. */
  variant?: string | null
  decided_at: string
  /** Автор — псевдоним. */
  author: string
  /** Причина или основание: код и текст. */
  reason: { code?: string | null; text: string } | null
  /** Параметры подписи записи (метод доп. проверки, номер разрешения…). */
  params?: Record<string, string | number>
  signature: RecordSignature
  /** Номер критического действия (AD-28). */
  ca_id?: string | null
  /** Решение исправляет прежнее — только новой записью (FR-146). */
  corrects?: { event_id: string; reason: string } | null
  /** Решение принято до более новой версии вывода системы — пересмотрите (FR-32). */
  before_revision?: boolean
}

/** Изоляция изделия (FR-55): в системе и физически. */
export interface Isolation {
  isolated_at: string
  /** Физически перемещено в изолятор (подтверждение с терминала или от мастера). */
  physically_moved: boolean
}

/** Карточка несоответствия (FR-51, PRD §3a «Контролёр качества»). */
export interface NcCard {
  nc_id: string
  /** Номер для людей. */
  number: string
  item_id: string
  item_label: string
  /** Операция и шаг процесса. */
  step_key?: string | null
  operation_label?: string | null
  /** Статус «по изделию». */
  status_by_item: NcByItemStatus
  /** Подписи маршрута решения «есть из нужно» — для `awaiting_approvals`. */
  approvals?: { have: number; need: number } | null
  /** Статус «системное расследование» — закрытие по изделию его не закрывает. */
  investigation: NcInvestigationStatus
  /** Статус исходного сигнала. */
  signal_status: SignalStatus
  /** Итоговый статус — оси статуса изделия. */
  item_statuses: ItemStatuses
  /** Срок решения по изоляции (FR-55); null — срока нет. */
  decision_due_at: string | null
  isolation?: Isolation | null
  /** Следующий шаг — для кнопки «Принять — передать на ‹шаг›». */
  next_step_label?: string | null
  /** Операция, на которую вернуть при переделке. */
  rework_operation_label?: string | null
  /** Доработки зоны против лимита (FR-18). */
  rework?: { zone_label: string; used: number; limit: number } | null
  /** Изделие уже обрабатывалось — «вернуть поставщику» недопустимо (FR-53). */
  item_processed: boolean
  signal: SourceSignal
  /** Требование; null — требования нет. */
  requirement: Requirement | null
  circumstances: NcCircumstances
  /** Версии вывода системы по возрастанию `version`. */
  conclusions: SystemConclusion[]
  similar_cases: SimilarCase[]
  /** Решения людей по времени. */
  decisions: HumanDecision[]
}

// ─────────────────────────────── очередь ───────────────────────────────

/** Что ждёт решения контролёра (PRD §3a): точка предъявления, сигнал, изолированное изделие. */
export type QueueEntryKind = 'presentation_point' | 'signal_review' | 'isolated_item'

/** Строка очереди «Ждут моего решения». */
export interface QueueEntry {
  entry_id: string
  kind: QueueEntryKind
  /** Куда ведёт строка: несоответствие или изделие. */
  ref: DrillRef
  item_id: string
  item_label: string
  nc_id?: string | null
  /** Что это — вид дефекта или точка предъявления, текстом сервера. */
  title: string
  /** Тяжесть; для точки предъявления — `unknown`, если не применима. */
  severity: Severity
  /** Ранг риска от сервера: чем больше, тем выше в очереди (AD-21: считает сервер). */
  risk_rank: number
  /** Состояние качества изделия — чтобы «оценка невозможна» не выглядела как «годно». */
  quality: QualityStatus
  /** Срок решения; null — срока нет. */
  due_at: string | null
  /** Когда поступило. */
  received_at: string
  /** Номер предъявления (для точки предъявления). */
  presentation_no?: number | null
  /** Изолировано в системе, физически не перемещено (FR-55). */
  not_moved?: boolean
}

// ──────────────────────── разрешения на отклонение ────────────────────────

/** Статус разрешения на отклонение — словарь `statuses.concession`. */
export type ConcessionStatus = 'draft' | 'on_approval' | 'active' | 'exhausted' | 'expired' | 'revoked'

/** Разрешение на отклонение для выбора в панели решений (FR-54). */
export interface ConcessionOption {
  concession_id: string
  number: string
  /** Пункт КД или ТУ. */
  clause: string
  /** Область действия: изделия или диапазон номеров. */
  scope: string
  limit: number
  used: number
  valid_until: string
  status: ConcessionStatus
  /** Покрывает это изделие (область действия); вычисляет сервер. */
  applies_to_item: boolean
}

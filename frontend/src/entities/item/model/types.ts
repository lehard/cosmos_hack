/**
 * Входные данные паспорта изделия (эпик 11; FR-42, FR-43, FR-45, FR-46, FR-140;
 * кейс «история изделия»).
 *
 * Поля названы как в контракте (contracts/events: конверт `common/envelope`,
 * семейства `item`, `inspection`, `quality`, `decision`, `document`) и в словаре
 * статусов (contracts/statuses.yaml): когда операция чтения паспорта появится в
 * contracts/openapi.yaml, сгенерированные типы подставляются сюда без
 * переименований. Перечисления берутся из сгенерированных типов событий
 * (shared/contracts/events.ts), а не переписываются руками.
 *
 * Паспорт — проекция журнала (AD-2): интерфейс ничего не вычисляет, кроме
 * показа. Смысл — только контрактный (NFR-UI-4): «оценка невозможна» ≠ «годно»,
 * «заблокировано» ≠ «признано дефектным», ручная отметка ≠ данные станка.
 */
import type {
  DecisionPresentationResolvedV1,
  DocumentSignatureRecordedV1,
  DocumentVersionDraftedV1,
  EventEnvelopeV1,
  ItemCarrierAppliedV1,
  ItemCarrierVerifiedV1,
} from '@/shared/contracts/events'
import type { statusAxes, statusDictionaries } from '@/shared/api/generated/statuses'

// ───────────────────────────── статусы (AD-30) ─────────────────────────────

/** Значение оси «Положение в процессе». */
export type PositionStatus = keyof typeof statusAxes.position.values
/** Значение оси «Состояние качества». */
export type QualityStatus = keyof typeof statusAxes.quality.values
/** Значение оси «Решение по изделию». */
export type DispositionStatus = keyof typeof statusAxes.disposition.values
/** Значение оси «Сдерживание». */
export type ContainmentStatus = keyof typeof statusAxes.containment.values
/** Значение оси «Учёт в 1С». */
export type ErpAccountingStatus = keyof typeof statusAxes.erp_accounting.values
/** Значение оси «Статус в инциденте». */
export type IncidentItemStatus = keyof typeof statusAxes.incident.values
/** Сводный статус изделия для списков (не ось, вычисляет сервер). */
export type ItemSummaryStatus = keyof typeof statusDictionaries.item_summary.values

/**
 * Пять осей статуса изделия (PRD §3b); шестая — статус в инциденте — у каждого
 * инцидента своя, поэтому отдельным списком `incidents`.
 */
export interface ItemStatuses {
  position: PositionStatus
  quality: QualityStatus
  disposition: DispositionStatus
  containment: ContainmentStatus
  erp_accounting: ErpAccountingStatus
}

/** Статус изделия в одном инциденте (FR-62). */
export interface ItemIncidentStatus {
  incident_id: string
  /** Подпись инцидента для людей. */
  label: string
  status: IncidentItemStatus
}

// ───────────────────────── источник и подпись факта ─────────────────────────

/** Вид источника факта (конверт `source_kind`, FR-140). */
export type SourceKind = NonNullable<EventEnvelopeV1['source_kind']>
/** Надёжность факта по источнику (конверт `reliability`, FR-140). */
export type Reliability = NonNullable<EventEnvelopeV1['reliability']>
/** Способ подписи (`document.signature.recorded.method`). */
export type SignatureMethod = DocumentSignatureRecordedV1['method']

/**
 * Результат автоматической проверки подписи записи (FR-68): подпись,
 * действительность сертификата на момент подписания, отзыв, полномочие.
 * `key_unavailable` — проверить нельзя (≠ «действительна»); `not_checked` —
 * проверка ещё не выполнена.
 */
export type SignatureVerification = 'valid' | 'invalid' | 'cert_revoked' | 'no_authority_at_signing' | 'key_unavailable' | 'not_checked'

/** Подпись записи паспорта: кто, как, каким уровнем и что показала проверка (FR-42, FR-68, FR-139). */
export interface RecordSignature {
  /** Итог проверки. */
  verification: SignatureVerification
  /** Способ подписи. */
  method: SignatureMethod
  /** Уровень подписи 0–3 (AD-13). */
  level: number
  /** Подписант — псевдоним человека или id устройства. */
  signer: string | null
  /** Цифровое клеймо контролёра, если ставилось (FR-145). */
  stamp_id?: string | null
  /** Заверитель бумажной подписи (AD-43). */
  attested_by?: string | null
  /** Учётный номер бумажного оригинала в архиве ОТК. */
  paper_original_no?: string | null
  /** Адрес скана в хранилище материалов. */
  scan_address?: string | null
}

// ───────────────────────────── записи паспорта ─────────────────────────────

/**
 * Запись паспорта — строка журнала по изделию (FR-42): факт, вывод системы,
 * решение человека или документ. Вид слоя интерфейс берёт из каталога типов
 * (`eventCatalog[event_type].kind`), а не из данных.
 */
export interface PassportRecord {
  /** `event_id` записи журнала. */
  event_id: string
  /** Тип записи из каталога (`семейство.сущность.действие`). */
  event_type: string
  /**
   * Уточнение внутри типа, если от него зависит текст: исход контроля
   * (`outcome`), решение на точке предъявления (`resolution`), вариант решения
   * по изделию (`disposition`), способ подписи (`method`).
   */
  variant?: string | null
  /** Позиция в основной цепочке журнала — ссылка в журнал событий (FR-43). */
  seq: number
  /** Время возникновения (AD-37). */
  occurred_at: string
  /** Время записи в журнал. */
  recorded_at: string
  /** Источник (`source_id` конверта): устройство, терминал, шлюз, `server`. */
  source_id: string
  /** Вид источника факта; у решений и выводов может отсутствовать. */
  source_kind?: SourceKind | null
  /** Надёжность факта по источнику. */
  reliability?: Reliability | null
  /** Автор — псевдоним человека (для решений и ручного ввода); null — не человек или неизвестно. */
  author?: string | null
  /** Ключ шага процесса (`step_key`). */
  step_key?: string | null
  /** Зоны изделия, к которым относится запись. */
  zone_ids?: string[]
  /** Параметры подписи записи, уже приведённые проекцией к тексту (метод, зона, решение…). */
  params?: Record<string, string | number>
  /** Подпись и её проверка. */
  signature: RecordSignature
  /** Запись исправляет прежнюю (FR-122): её `event_id` и обоснование. */
  corrects?: { event_id: string; reason: string } | null
  /** Запись исправлена более поздней — `event_id` исправляющей. */
  corrected_by?: string | null
  /** Вывод системы пересмотрен из-за записи (конверт `reaction.revised_due_to`). */
  revised_due_to?: string | null
  /** Номер критического действия `CA-‹n›` (AD-28), если запись — критическое действие. */
  ca_id?: string | null
}

/**
 * Изменение паспорта (FR-43): отдельный журнал изменений, связанный с глобальным
 * журналом событий через `seq` и `event_id`. Исправление — только новой строкой.
 */
export interface PassportChange {
  /** Номер изменения в журнале изменений паспорта. */
  change_no: number
  /** Запись журнала, которая вызвала изменение. */
  event_id: string
  /** Её позиция в основной цепочке. */
  seq: number
  /** Время записи. */
  recorded_at: string
  /** Какая ось статуса изменилась; null — изменилось другое поле паспорта. */
  axis: keyof ItemStatuses | null
  /** Поле паспорта, если это не ось (носитель, зона, документ…). */
  field?: string | null
  /** Было (код значения оси или текст поля); null — не было. */
  before: string | null
  /** Стало. */
  after: string | null
  /** Кто: псевдоним человека; null — движок по фактам или правило. */
  author: string | null
  /** Правило, если изменение сделала реакция движка (конверт `reaction.rule_id`). */
  rule_id?: string | null
  /** Исправляемое изменение: `event_id` исправляемой записи. */
  corrects_event_id?: string | null
  /** Основание изменения (текст причины). */
  reason?: string | null
}

// ───────────────────────── носители, зоны, генеалогия ─────────────────────────

/** Тип носителя идентификатора (AD-16). */
export type CarrierType = ItemCarrierAppliedV1['carrier_type']

/** Носитель идентификатора изделия (`item.carrier.*`). */
export interface ItemCarrier {
  carrier_type: CarrierType
  /** Значение носителя. */
  value: string
  /** Временный носитель (бирка до DPM). */
  is_temporary: boolean
  /** Нанесён или уже снят. */
  state: 'applied' | 'removed'
  /** Когда нанесён. */
  applied_at: string
  /** Когда снят. */
  removed_at?: string | null
  /** Зона нанесения по КД. */
  zone_id?: string | null
  /** Последнее считывание: прочитан / нечитаем / не совпал; null — не считывался. */
  last_read?: ItemCarrierVerifiedV1['read_outcome'] | null
}

/** Статус зоны изделия (FR-46): проверена / устарела после вмешательства / закрыта. */
export type ZoneStatus = 'checked' | 'stale_after_intervention' | 'closed'

/** Зона изделия (FR-46). */
export interface ItemZone {
  zone_id: string
  /** Подпись зоны (участок шва, отверстие). */
  label: string
  /** Статус; null — ещё не проверялась (≠ «проверена»). */
  status: ZoneStatus | null
  /** Последний результат контроля зоны. */
  last_inspection?: { event_id: string; outcome: string; occurred_at: string } | null
  /** Доработки зоны: сколько было из лимита нормативного слоя (FR-18). */
  rework?: { used: number; limit: number } | null
  /** Открыто вмешательство (FR-21). */
  intervention_open?: boolean
}

/** Узел генеалогии (FR-45): экземпляр или партия. */
export interface GenealogyNode {
  /** Экземпляр — `item_id`; для партионного компонента null. */
  item_id: string | null
  /** Партия, садка, плавка — для партионного компонента и корней. */
  lot_id?: string | null
  /** Подпись: номер детали, обозначение, номер партии. */
  label: string
  /** Тип по спецификации. */
  item_type_id?: string | null
  /** Позиция по спецификации. */
  position?: string | null
  /** Количество (партионный компонент). */
  quantity?: number | null
  /** Сводный статус узла, если это экземпляр. */
  summary?: ItemSummaryStatus | null
}

/** Генеалогия изделия (FR-45). */
export interface ItemGenealogy {
  /** Куда входит (сборки верхнего уровня). */
  up: GenealogyNode[]
  /** Из чего состоит (компоненты). */
  down: GenealogyNode[]
  /** Партии материала и покупных — корни генеалогии. */
  lots: GenealogyNode[]
}

// ─────────────────────────────── документы ───────────────────────────────

/** Статус документа (словарь `statuses.document`). */
export type DocumentStatus = 'draft' | 'on_route' | 'route_closed' | 'annulled'

/** Документ изделия (FR-65): собран из истории по шаблону шага. */
export interface PassportDocument {
  document_id: string
  version: number
  /** Шаблон `‹id›@‹версия›`. */
  template_ref: DocumentVersionDraftedV1['template_ref']
  /** Вид документа — ключ `documents.types.*` в snake_case (`traveler`, `scrap_act`…). */
  doc_type: string
  status: DocumentStatus
  /** Отпечаток (`doc_digest`). */
  doc_digest: string
  /** Когда сформирована версия. */
  drafted_at: string
  /** Подписи версии. */
  signatures: RecordSignature[]
  /** Собран из истории изделия, а не введён вручную. */
  generated_from_history: boolean
}

/** Предъявление изделия на точке (FR-19). */
export interface ItemPresentation {
  step_key: string
  presentation_no: number
  presented_to: 'qc' | 'customer_representative'
  presented_at: string
  /** Решение, если принято. */
  resolution?: DecisionPresentationResolvedV1['resolution'] | null
}

// ─────────────────────────────── паспорт ───────────────────────────────

/** Паспорт изделия на момент (FR-42, AD-22). */
export interface ItemPassport {
  /** Внутренний ID `код_предприятия:локальный_id`. */
  item_id: string
  /** Номер детали для людей. */
  label: string
  /** Тип изделия (обозначение по КД). */
  item_type_id: string
  /** Ревизия КД и ТП на момент запуска. */
  item_revision: string
  /** Версия процесса, закреплённая при запуске (подпись для людей). */
  process_version: string
  /** Ревизия нормативного слоя при запуске. */
  normative_rev: string
  /** Сборочная единица. */
  is_assembly: boolean
  /** Оси статуса. */
  statuses: ItemStatuses
  /** Сводный статус (для списков и заголовка). */
  summary: ItemSummaryStatus
  /** Статус в каждом инциденте. */
  incidents: ItemIncidentStatus[]
  /** Идентификация под сомнением (AD-16) — изделие изолируется до повторной идентификации. */
  identification_questioned?: boolean
  /** Обработка изделия остановлена из-за ошибки (ops.processing.failed). */
  processing_stopped?: boolean
  carriers: ItemCarrier[]
  zones: ItemZone[]
  genealogy: ItemGenealogy
  /** Все записи паспорта: факты, выводы, решения, документы. */
  records: PassportRecord[]
  /** Журнал изменений паспорта (FR-43). */
  changes: PassportChange[]
  documents: PassportDocument[]
  presentations: ItemPresentation[]
  /** Сколько документов собрано из истории — «вручную не понадобилось» (FR-65). */
  documents_from_history: number
}

/**
 * Правила показа паспорта изделия (FR-42, FR-43, FR-45, FR-46, FR-140,
 * NFR-UI-4): чистые функции над ответами API — текст и слой записи, пометка
 * источника, итог проверки подписи, состояние экрана, разбор генеалогии.
 *
 * Интерфейс ничего не пересчитывает (AD-21): слой записи — поле `kind`, текст —
 * `summary` сервера (нет — название типа из каталога, нет и там — UNKNOWN(тип)),
 * тон — из словаря статусов, подписи — ключами vue-i18n. Код, которого нет в
 * таблице, не подменяется похожим.
 */
import { statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import type { WidgetDataState } from '@/shared/config/widget'
import { eventCatalog } from '@/shared/contracts/catalog'
import type {
  GenealogyNode,
  ItemCarrierCarrierType,
  ItemCarrierState,
  ItemDocumentRefStatus,
  ItemGenealogy,
  ItemPassportIdentification,
  ItemStatuses,
  ItemZone,
  PassportEntry,
  RecordLayer,
  RecordSignature,
  Reliability,
  SignatureCheck,
  SignatureClass,
  SourceKind,
  SourcedRecord,
} from './types'

/**
 * Текст кода по таблице или UNKNOWN(код) — неизвестный код не подменяется
 * похожим (NFR-UI-4); отсутствующий — «неизвестно».
 * @param table — таблица «код → ключ»
 * @param code — код контракта
 * @param t — функция текстов
 */
export function codeText(table: Readonly<Record<string, string>>, code: string | null | undefined, t: (key: string) => string): string {
  if (code == null) return t('common.words.unknown')
  const key = table[code]
  return key ? t(key) : `UNKNOWN(${code})`
}

const catalog = eventCatalog as Record<string, { title: string; kind: string; critical: boolean } | undefined>

// ─────────────────────────────── текст и слой ───────────────────────────────

/**
 * Текст записи: краткое содержание от сервера; нет — название типа из каталога;
 * нет и там — UNKNOWN(тип).
 */
export function entryText(e: Pick<PassportEntry, 'summary' | 'event_type'>): string {
  if (e.summary) return e.summary
  return catalog[e.event_type]?.title ?? `UNKNOWN(${e.event_type})`
}

/** Слои в порядке фильтра. */
export const RECORD_LAYERS: readonly RecordLayer[] = ['fact', 'reaction', 'decision', 'service']

/**
 * Ключ текста слоя (FR-51, AD-2): факт / вывод системы / решение человека /
 * служебная запись — различимы на экране.
 */
export const LAYER_TEXT: Record<RecordLayer, string> = {
  fact: 'timeline.layer.fact',
  reaction: 'timeline.layer.conclusion',
  decision: 'timeline.layer.decision',
  service: 'widgets.passport.layers.service',
}

/** Критическое ли действие по каталогу (AD-28). */
export const isCriticalType = (eventType: string): boolean => catalog[eventType]?.critical === true

/** Записи по времени возникновения, при равенстве — по `seq` (AD-37). */
export const sortEntries = <T extends Pick<PassportEntry, 'occurred_at' | 'seq'>>(entries: readonly T[]): T[] =>
  [...entries].sort((a, b) => a.occurred_at.localeCompare(b.occurred_at) || a.seq - b.seq)

// ────────────────────────────── пометка источника ──────────────────────────────

/** Ключ текста вида источника (FR-140). */
export const SOURCE_KIND_TEXT: Record<SourceKind, string> = {
  manual_entry: 'timeline.sourceKind.manual',
  import: 'timeline.sourceKind.import',
  machine: 'timeline.sourceKind.machine',
  sensor: 'timeline.sourceKind.sensor',
  camera: 'timeline.sourceKind.camera',
  external_system: 'timeline.sourceKind.external',
}

/** Ключ текста надёжности факта (FR-140). */
export const RELIABILITY_TEXT: Record<Reliability, string> = {
  high: 'widgets.passport.reliability.high',
  medium: 'widgets.passport.reliability.medium',
  low: 'widgets.passport.reliability.low',
  unknown: 'widgets.passport.reliability.unknown',
}

/** Пометка источника у записи. */
export interface SourceMark {
  /** Код пометки — для атрибута `data-source` и тестов. */
  code: SourceKind | 'system' | 'person' | 'unknown'
  /** Ключ текста. */
  key: string
  /** Ключ текста надёжности; null — источник не сообщил. */
  reliabilityKey: string | null
  /** Ручной ввод или импорт — не данные оборудования (FR-140). */
  manual: boolean
}

/**
 * Пометка источника (FR-140): у факта — вид источника, у вывода системы —
 * «вывод системы», у решения — «решение человека». Ручная отметка остаётся
 * ручной; факт без вида источника — «источник неизвестен», а не «станок»
 * (FR-123: не додумывать).
 */
export function sourceMark(r: SourcedRecord): SourceMark {
  const reliabilityKey = r.reliability ? RELIABILITY_TEXT[r.reliability] : null
  if (r.source_kind) {
    return { code: r.source_kind, key: SOURCE_KIND_TEXT[r.source_kind], reliabilityKey, manual: r.source_kind === 'manual_entry' || r.source_kind === 'import' }
  }
  if (r.kind === 'reaction' || r.kind === 'service') return { code: 'system', key: 'timeline.sourceKind.system', reliabilityKey, manual: false }
  if (r.kind === 'decision') return { code: 'person', key: 'timeline.layer.decision', reliabilityKey, manual: false }
  return { code: 'unknown', key: 'widgets.passport.source.unknown', reliabilityKey, manual: false }
}

// ─────────────────────────────── проверка подписи ───────────────────────────────

/**
 * Текст и тон итога проверки подписи (FR-68). «Не проверяемо» и «не
 * проверялась» ≠ «действительна»: зелёная пометка — только у `valid`.
 */
export const SIGNATURE_CHECK: Record<SignatureCheck, { key: string; tone: StatusTone }> = {
  valid: { key: 'audit.signatureCheck.valid', tone: 'success' },
  rejected: { key: 'audit.signatureCheck.invalid', tone: 'danger' },
  not_verifiable: { key: 'widgets.passport.signature.notVerifiable', tone: 'attention' },
  unchecked: { key: 'widgets.passport.signature.unchecked', tone: 'neutral' },
}

/** Ключ текста класса происхождения подписи (AD-2; FR-139 — способ подписи в журнале). */
export const SIGNATURE_CLASS_TEXT: Record<SignatureClass, string> = {
  device: 'widgets.passport.signature.class.device',
  personal: 'widgets.passport.signature.class.personal',
  paper: 'widgets.passport.signature.class.paper',
  partner: 'widgets.passport.signature.class.partner',
  server_attested: 'widgets.passport.signature.class.serverAttested',
  scenario: 'widgets.passport.signature.class.scenario',
  genesis: 'widgets.passport.signature.class.genesis',
}

/** Цвет пометки проверки подписи — из словаря статусов (AD-30). */
export const signatureColor = (s: Pick<RecordSignature, 'check'>): string => statusPalette[SIGNATURE_CHECK[s.check].tone]

// ─────────────────────────────── статусы и экран ───────────────────────────────

/**
 * Состояние данных для рамки (AD-21, NFR-UI-4) — только по оси «Состояние
 * качества»: сигнал или подтверждённое несоответствие → «признак дефекта»;
 * «оценка невозможна» → своё состояние (не «годно»); остальное → норма.
 * Сдерживание («заблокировано») и изоляция на состояние не влияют: блок ≠
 * признание дефекта, снятие блока ≠ годность.
 */
export function qualityState(statuses: Pick<ItemStatuses, 'quality'>): WidgetDataState {
  if (statuses.quality === 'signal' || statuses.quality === 'nonconforming') return 'defect_indication'
  if (statuses.quality === 'unable_to_assess') return 'unable_to_assess'
  return 'normal'
}

/** Порядок осей статуса на экране (PRD §3b). */
export const STATUS_AXES_ORDER: readonly (keyof ItemStatuses)[] = ['position', 'quality', 'disposition', 'containment', 'erp_accounting']

/** Ключ названия оси статуса. */
export const AXIS_TEXT: Record<keyof ItemStatuses, string> = {
  position: 'widgets.passport.axes.position',
  quality: 'widgets.passport.axes.quality',
  disposition: 'widgets.passport.axes.disposition',
  containment: 'widgets.passport.axes.containment',
  erp_accounting: 'widgets.passport.axes.erpAccounting',
}

/** Блок изделия или партии — сдерживание, а не признание дефекта. */
export const isHeld = (s: Pick<ItemStatuses, 'containment'>): boolean => s.containment === 'item_hold' || s.containment === 'lot_hold'

/** Ключ текста уровня идентификации (AD-16, FR-34). */
export const IDENTIFICATION_TEXT: Record<ItemPassportIdentification, string> = {
  unique: 'widgets.passport.identification.unique',
  probable: 'widgets.passport.identification.probable',
  ambiguous: 'widgets.passport.identification.ambiguous',
  unidentified: 'widgets.passport.identification.unidentified',
}

/** Идентификация под сомнением — изоляция до повторной идентификации человеком (AD-16). */
export const identificationDoubtful = (level: ItemPassportIdentification): boolean => level === 'ambiguous' || level === 'unidentified'

// ─────────────────────────── зоны, носители, документы ───────────────────────────

/** Статус зоны, который можно утверждать по ответу сервера. */
export type ZoneStatus = 'closed' | 'stale_after_intervention'

/**
 * Статус зоны (FR-46) из того, что сообщил сервер: открыто вмешательство →
 * «устарела после вмешательства»; доступ закрыт → «закрыта»; иначе null —
 * сервер не сообщил «проверена», и экран этого не утверждает.
 */
export function zoneStatus(z: Pick<ItemZone, 'closed' | 'open_intervention'>): ZoneStatus | null {
  if (z.open_intervention) return 'stale_after_intervention'
  if (z.closed) return 'closed'
  return null
}

/** Ключ текста статуса зоны. */
export const ZONE_STATUS_TEXT: Record<ZoneStatus, string> = {
  closed: 'statuses.zone.closed',
  stale_after_intervention: 'statuses.zone.staleAfterIntervention',
}

/** Ключ текста типа носителя (AD-16). */
export const CARRIER_TYPE_TEXT: Record<ItemCarrierCarrierType, string> = {
  dpm_datamatrix: 'timeline.bindingMethod.dataMatrix',
  tag_qr: 'timeline.bindingMethod.tag',
  container_cell: 'widgets.passport.carrier.type.containerCell',
  route_card: 'timeline.bindingMethod.routeCard',
  post_context: 'timeline.bindingMethod.stationContext',
  manual_entry: 'timeline.bindingMethod.manual',
  internal_id: 'widgets.passport.carrier.type.internalId',
}

/** Ключ текста состояния носителя. */
export const CARRIER_STATE_TEXT: Record<ItemCarrierState, string> = {
  applied: 'widgets.passport.carrier.state.applied',
  verified: 'widgets.passport.carrier.state.verified',
  unreadable: 'widgets.passport.carrier.state.unreadable',
  removed: 'widgets.passport.carrier.state.removed',
  replaced: 'widgets.passport.carrier.state.replaced',
}

/** Ключ текста статуса документа (`statuses.document`). */
export const DOCUMENT_STATUS_TEXT: Record<ItemDocumentRefStatus, string> = {
  drafted: 'statuses.document.draft',
  in_route: 'statuses.document.onRoute',
  closed: 'statuses.document.routeClosed',
  annulled: 'statuses.document.annulled',
}

// ─────────────────────────────── генеалогия ───────────────────────────────

/** Генеалогия, разложенная для показа (FR-45). */
export interface GenealogyView {
  /** Куда входит изделие — от ближайшей сборки вверх. */
  up: GenealogyNode[]
  /** Из чего состоит. */
  down: GenealogyNode[]
  /** Партии, садки, плавки — корни. */
  lots: GenealogyNode[]
  /** Выписки паспорта партнёров (AD-19). */
  extracts: GenealogyNode[]
}

/**
 * Разложить узлы генеалогии: «куда входит» — цепочка `parent_ref` от самого
 * изделия вверх; «из чего состоит» — узлы с `parent_ref` = изделие; партии и
 * выписки — отдельно.
 * @param g — ответ `item.genealogy.read`
 */
export function splitGenealogy(g: ItemGenealogy): GenealogyView {
  const byRef = new Map(g.nodes.map((n) => [n.ref, n]))
  const up: GenealogyNode[] = []
  const seen = new Set<string>([g.item_id])
  let parent = byRef.get(g.item_id)?.parent_ref ?? null
  while (parent && !seen.has(parent)) {
    seen.add(parent)
    const node = byRef.get(parent)
    if (!node) break
    up.push(node)
    parent = node.parent_ref ?? null
  }
  return {
    up,
    down: g.nodes.filter((n) => n.kind !== 'partner_extract' && n.ref !== g.item_id && n.parent_ref === g.item_id),
    lots: g.nodes.filter((n) => n.kind === 'lot' && n.parent_ref !== g.item_id),
    extracts: g.nodes.filter((n) => n.kind === 'partner_extract'),
  }
}

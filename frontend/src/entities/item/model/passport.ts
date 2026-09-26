/**
 * Правила показа паспорта изделия (FR-42, FR-43, FR-140, NFR-UI-4): чистые
 * функции над входными данными — какой текст у записи, какой слой, какая
 * пометка источника, что показала проверка подписи, в каком состоянии экран.
 *
 * Интерфейс ничего не пересчитывает (AD-21): слой записи — из каталога типов
 * (contracts/events/catalog.yaml), тон — из словаря статусов, текст — ключом
 * vue-i18n. Тип записи, которого нет в таблице, не подменяется похожим: берётся
 * название из каталога, а если типа нет и там — UNKNOWN(тип).
 */
import { statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import type { WidgetDataState } from '@/shared/config/widget'
import { eventCatalog } from '@/shared/contracts/catalog'
import type { ItemPassport, ItemStatuses, PassportRecord, RecordSignature, Reliability, SignatureMethod, SignatureVerification, SourceKind } from './types'

// ─────────────────────────────── текст записи ───────────────────────────────

/** Текст записи: ключ vue-i18n или название типа из каталога. */
export interface RecordText {
  /** Ключ текста; null — берём `title`. */
  key: string | null
  params: Record<string, string | number>
  /** Название типа из каталога — если ключа нет; null — типа нет и в каталоге. */
  title: string | null
  /** Исходный тип — для UNKNOWN(…). */
  eventType: string
}

/** Уточнение → ключ текста; `*` — любое уточнение. */
const TEXTS: Record<string, Record<string, string>> = {
  'item.item.registered': { '*': 'timeline.item.itemRegistered' },
  'item.carrier.applied': { '*': 'timeline.item.itemMarked' },
  'item.assembly.recorded': { '*': 'timeline.item.assemblyRecorded' },
  'item.presentation.recorded': { '*': 'timeline.item.itemPresented' },
  'item.release.recorded': { '*': 'timeline.item.itemReleased' },
  'item.intervention.opened': { '*': 'timeline.operation.interventionOpened' },
  'item.intervention.closed': { '*': 'timeline.operation.interventionClosed' },
  // FR-36: три исхода контроля — три разных текста; «не обнаружено» ≠ «годно».
  'inspection.result.recorded': {
    defect_indicated: 'timeline.inspection.resultDefectFound',
    no_defect_indicated: 'timeline.inspection.resultNoDefectFound',
    unable_to_assess: 'timeline.inspection.resultUnableToAssess',
  },
  'quality.signal.raised': { '*': 'timeline.quality.signalReceived' },
  'quality.observation.linked': { '*': 'timeline.quality.observationLinked' },
  'quality.inspection.missing': { '*': 'timeline.quality.inspectionMissing' },
  'quality.escape.recorded': { '*': 'timeline.quality.escapeRecorded' },
  'operation.run.started': { '*': 'timeline.operation.runStarted' },
  'operation.run.paused': { '*': 'timeline.operation.runPaused' },
  'operation.run.resumed': { '*': 'timeline.operation.runResumed' },
  'operation.run.finished': { '*': 'timeline.operation.runFinished' },
  'operation.movement.sent': { '*': 'timeline.operation.movementSent' },
  'operation.movement.received': { '*': 'timeline.operation.movementReceived' },
  'decision.nonconformity.drafted': { '*': 'timeline.decision.ncDraftCreated' },
  // «Заблокировано» — сдерживание, а не признание дефекта (NFR-UI-4).
  'decision.containment.applied': { '*': 'timeline.decision.holdAppliedByRule' },
  'decision.containment.set': { '*': 'widgets.passport.records.containmentSet' },
  'decision.containment.released': { '*': 'timeline.decision.holdReleased' },
  'decision.nonconformity.confirmed': { '*': 'timeline.decision.nonconformityConfirmed' },
  'decision.signal.rejected': { '*': 'timeline.decision.signalRejected' },
  'decision.recheck.requested': { '*': 'timeline.decision.recheckRequested' },
  'decision.item.isolated': { '*': 'timeline.decision.itemIsolated' },
  'decision.disposition.set': { '*': 'timeline.decision.dispositionSet' },
  'decision.disposition.applied': { '*': 'widgets.passport.records.dispositionByRule' },
  'decision.disposition.verified': { '*': 'timeline.decision.dispositionVerified' },
  // «Принято по разрешению на отклонение» ≠ «принято» (FR-53).
  'decision.presentation.resolved': {
    accept: 'timeline.decision.gatePassed',
    accept_with_concession: 'widgets.passport.records.gateAcceptedWithConcession',
    reject: 'timeline.decision.gateReturned',
    insufficient_data: 'widgets.passport.records.gateInsufficientData',
  },
  'decision.nonconformity.closed': { '*': 'timeline.decision.itemClosed' },
  'decision.concession.revoked': { '*': 'timeline.decision.concessionRevoked' },
  'document.version.drafted': { '*': 'timeline.document.drafted' },
  'document.signature.recorded': { paper: 'timeline.document.signedOnPaper', '*': 'timeline.document.signed' },
  'document.route.closed': { '*': 'timeline.document.routeClosed' },
  'document.version.annulled': { '*': 'timeline.document.annulled' },
  'genealogy.link.added': { '*': 'timeline.genealogy.linkAdded' },
  'binding.link.resolved': { '*': 'timeline.binding.linkResolved' },
  'obligation.due.set': { '*': 'timeline.obligation.dueSet' },
  'obligation.due.cleared': { '*': 'timeline.obligation.dueCleared' },
  'erp.posting.requested': { '*': 'timeline.integration.postingSent' },
  'erp.posting.responded': { '*': 'timeline.integration.ack' },
  'ops.processing.failed': { '*': 'timeline.ops.processingFailed' },
  'incident.item.assessed': { '*': 'timeline.incident.itemAssessed' },
  // Обстоятельства операции в карточке несоответствия (FR-148): оборудование и
  // действия исполнителя — обстоятельство, а не вина.
  'equipment.state.changed': {
    cycle_started: 'timeline.equipment.cycleStarted',
    cycle_finished: 'timeline.equipment.cycleFinished',
    warning: 'timeline.equipment.warning',
    fault: 'timeline.equipment.alarm',
    '*': 'timeline.equipment.stateChanged',
  },
  'equipment.deviation.detected': {
    out_of_setpoint: 'timeline.equipment.parameterOutOfRange',
    manual_override: 'timeline.equipment.manualOverride',
    tool_life_warning: 'timeline.equipment.toolLife',
    unplanned_program_change: 'timeline.equipment.programChanged',
    alarm: 'timeline.equipment.alarm',
    '*': 'timeline.equipment.warning',
  },
  'equipment.program.changed': { '*': 'timeline.equipment.programChanged' },
  'equipment.tool.changed': { '*': 'timeline.equipment.toolLife' },
  'equipment.cycle.summarized': { '*': 'timeline.equipment.parametersRecorded' },
  'operator.step.confirmed': { '*': 'timeline.operator.stepConfirmed' },
  'operator.mode.changed': { '*': 'timeline.operator.modeChange' },
  'operator.check.skipped': { '*': 'timeline.operator.checkSkipped' },
  'operator.override.performed': { '*': 'timeline.operator.manualIntervention' },
  'operator.deviation.reported': { '*': 'timeline.operator.deviationReported' },
  'operator.inspection.requested': { '*': 'timeline.operator.inspectionRequested' },
  // OperatorVision — только гипотеза о действии, текст это говорит.
  'operator.action.observed': { '*': 'timeline.operator.actionDetected' },
}

const catalog = eventCatalog as Record<string, { title: string; kind: string; critical: boolean } | undefined>

/**
 * Текст записи паспорта.
 * @param r — запись с типом, уточнением и параметрами
 */
export function describePassportRecord(r: Pick<PassportRecord, 'event_type' | 'variant' | 'params'>): RecordText {
  const byVariant = TEXTS[r.event_type]
  const key = byVariant ? (byVariant[r.variant ?? ''] ?? byVariant['*'] ?? null) : null
  return { key, params: r.params ?? {}, title: catalog[r.event_type]?.title ?? null, eventType: r.event_type }
}

// ──────────────────────────────── слой записи ────────────────────────────────

/**
 * Слой записи на экране (FR-51, «одна правда»): факт / вывод системы / решение
 * человека / обмен с внешней системой / служебная запись.
 */
export type RecordLayer = 'fact' | 'conclusion' | 'decision' | 'exchange' | 'service'

/** Все слои в порядке фильтра. */
export const RECORD_LAYERS: readonly RecordLayer[] = ['fact', 'conclusion', 'decision', 'exchange', 'service']

/** Семейства обмена с внешними системами (AD-18). */
const EXCHANGE_FAMILIES = new Set(['erp', 'mes', 'federation', 'cad'])

/**
 * Слой записи — по виду типа в каталоге (AD-2): `fact` → факт, `reaction` →
 * вывод системы, `decision` → решение человека, `service` → служебная;
 * семейства внешних систем — обмен.
 */
export function recordLayer(eventType: string): RecordLayer {
  if (EXCHANGE_FAMILIES.has(eventType.split('.')[0] ?? '')) return 'exchange'
  const kind = catalog[eventType]?.kind
  if (kind === 'fact') return 'fact'
  if (kind === 'reaction') return 'conclusion'
  if (kind === 'decision') return 'decision'
  return 'service'
}

/** Ключ текста слоя. */
export const LAYER_TEXT: Record<RecordLayer, string> = {
  fact: 'timeline.layer.fact',
  conclusion: 'timeline.layer.conclusion',
  decision: 'timeline.layer.decision',
  exchange: 'timeline.layer.exchange',
  service: 'widgets.passport.layers.service',
}

/** Критическое ли действие по каталогу (AD-28). */
export const isCriticalType = (eventType: string): boolean => catalog[eventType]?.critical === true

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

/** Пометка источника у записи: что за источник и насколько надёжен. */
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
 * Пометка источника записи (FR-140): у факта — вид источника из конверта, у
 * вывода системы — «вывод системы», у решения — «решение человека». Ручная
 * отметка остаётся ручной, даже если пришла с поста у станка.
 */
export function sourceMark(r: Pick<PassportRecord, 'event_type' | 'source_kind' | 'reliability'>): SourceMark {
  const reliabilityKey = r.reliability ? RELIABILITY_TEXT[r.reliability] : null
  if (r.source_kind) {
    return { code: r.source_kind, key: SOURCE_KIND_TEXT[r.source_kind], reliabilityKey, manual: r.source_kind === 'manual_entry' || r.source_kind === 'import' }
  }
  const layer = recordLayer(r.event_type)
  if (layer === 'conclusion' || layer === 'service') return { code: 'system', key: 'timeline.sourceKind.system', reliabilityKey, manual: false }
  if (layer === 'decision') return { code: 'person', key: 'timeline.layer.decision', reliabilityKey, manual: false }
  // Факт без вида источника — «неизвестен», а не «станок» (FR-123: не додумывать).
  return { code: 'unknown', key: 'widgets.passport.source.unknown', reliabilityKey, manual: false }
}

// ─────────────────────────────── проверка подписи ───────────────────────────────

/** Текст и тон результата проверки подписи (FR-68). */
export const SIGNATURE_CHECK: Record<SignatureVerification, { key: string; tone: StatusTone }> = {
  valid: { key: 'audit.signatureCheck.valid', tone: 'success' },
  invalid: { key: 'audit.signatureCheck.invalid', tone: 'danger' },
  cert_revoked: { key: 'audit.signatureCheck.certRevoked', tone: 'danger' },
  no_authority_at_signing: { key: 'audit.signatureCheck.noAuthorityAtSigning', tone: 'danger' },
  // «Проверить нельзя» ≠ «подпись действительна».
  key_unavailable: { key: 'errors.signing.keyUnavailable', tone: 'attention' },
  not_checked: { key: 'widgets.passport.signature.notChecked', tone: 'neutral' },
}

/** Ключ текста способа подписи (FR-139: в журнале фиксируется способ). */
export const SIGNATURE_METHOD_TEXT: Record<SignatureMethod, string> = {
  token_agent: 'widgets.passport.signature.method.tokenAgent',
  paper: 'widgets.passport.signature.method.paper',
  device: 'widgets.passport.signature.method.device',
  demo_signer: 'widgets.passport.signature.method.demoSigner',
}

/** Цвет пометки проверки подписи — из словаря статусов (AD-30). */
export const signatureColor = (s: Pick<RecordSignature, 'verification'>): string => statusPalette[SIGNATURE_CHECK[s.verification].tone]

/** Подпись проверена и действительна — единственный случай «зелёной» пометки. */
export const isSignatureValid = (s: Pick<RecordSignature, 'verification'>): boolean => s.verification === 'valid'

// ─────────────────────────────── состояние экрана ───────────────────────────────

/**
 * Состояние данных паспорта для рамки (AD-21, NFR-UI-4) — только по оси
 * «Состояние качества»:
 * - сигнал или подтверждённое несоответствие → «признак дефекта»;
 * - «оценка невозможна» → «оценка невозможна» (не «годно»);
 * - остальное → норма. Сдерживание («заблокировано») и изоляция на состояние
 *   не влияют: блок ≠ признание дефекта, снятие блока ≠ годность.
 */
export function qualityState(statuses: Pick<ItemStatuses, 'quality'>): WidgetDataState {
  if (statuses.quality === 'signal' || statuses.quality === 'nonconforming') return 'defect_indication'
  if (statuses.quality === 'unable_to_assess') return 'unable_to_assess'
  return 'normal'
}

/** Состояние данных паспорта. */
export const passportState = (p: Pick<ItemPassport, 'statuses'>): WidgetDataState => qualityState(p.statuses)

/** Записи паспорта по времени возникновения, при равенстве — по `seq` (AD-37). */
export const sortRecords = <T extends Pick<PassportRecord, 'occurred_at' | 'seq'>>(records: readonly T[]): T[] =>
  [...records].sort((a, b) => a.occurred_at.localeCompare(b.occurred_at) || a.seq - b.seq)

/** Порядок осей статуса на экране (PRD §3b). */
export const STATUS_AXES_ORDER: readonly (keyof ItemStatuses)[] = ['position', 'quality', 'disposition', 'containment', 'erp_accounting']

// ─────────────────────────────── строка записи ───────────────────────────────

/** Ключ названия статуса «Решение по изделию» — для параметра `{disposition}`. */
const DISPOSITION_KEY: Record<string, string> = {
  rework: 'statuses.disposition.rework',
  repair: 'statuses.disposition.repair',
  use_as_is: 'statuses.disposition.useAsIs',
  scrap: 'statuses.disposition.scrap',
  return_to_supplier: 'statuses.disposition.returnToSupplier',
}

type Translate = (key: string, params?: Record<string, unknown>) => string

/**
 * Строка записи для людей: текст по ключу, иначе название типа из каталога,
 * иначе UNKNOWN(тип). Вариант решения по изделию берётся кодом из уточнения и
 * называется словом словаря статусов — не «как прислал источник».
 * @param r — запись
 * @param t — функция текстов vue-i18n
 */
export function recordLabel(r: Pick<PassportRecord, 'event_type' | 'variant' | 'params'>, t: Translate): string {
  const x = describePassportRecord(r)
  const params: Record<string, unknown> = { ...x.params }
  if (r.event_type.startsWith('decision.disposition.') && r.variant && DISPOSITION_KEY[r.variant]) {
    params.disposition = t(DISPOSITION_KEY[r.variant]!)
  }
  if (x.key) return t(x.key, params)
  return x.title ?? `UNKNOWN(${x.eventType})`
}

/** Ключ названия оси статуса. */
export const AXIS_TEXT: Record<keyof ItemStatuses, string> = {
  position: 'widgets.passport.axes.position',
  quality: 'widgets.passport.axes.quality',
  disposition: 'widgets.passport.axes.disposition',
  containment: 'widgets.passport.axes.containment',
  erp_accounting: 'widgets.passport.axes.erpAccounting',
}

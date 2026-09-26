/**
 * Коды контракта → ключи текстов интерфейса (NFR-UI-3, NFR-UI-4). Каждый код
 * получает свой текст; код, которого нет в таблице, показывается как
 * UNKNOWN(код), а не подменяется похожим (`codeText`). Где в ru.json процессной
 * сессии нет точного текста для кода контракта, текст взят из ru.shell.json
 * (раздел `widgets.codes`), а не из соседнего по смыслу ключа: например,
 * «капиллярный» контроль не называется «неразрушающим вообще».
 */
import type { InspectionResultRecordedV1 } from '@/shared/contracts/events'
import type { Disposition, InspectionMethod, NcByItemStatus, ReactionOutcome, Severity, SignalStatus } from './types'

/** Метод контроля → текст (`inspection.method.*`). */
export const METHOD_TEXT: Record<InspectionMethod, string> = {
  camera: 'inspection.method.automatedVisual',
  cmm: 'inspection.method.cmm',
  radiography: 'inspection.method.xray',
  ultrasonic: 'inspection.method.ultrasonic',
  penetrant: 'widgets.codes.method.penetrant',
  leak_test: 'inspection.method.leakTest',
  torque: 'widgets.codes.method.torque',
  visual_human: 'inspection.method.visualHuman',
  supplier_documents: 'inspection.method.documents',
  laboratory: 'inspection.method.laboratory',
  other: 'widgets.codes.method.other',
}

/** Фаза контроля → текст. */
export const PHASE_TEXT: Record<InspectionResultRecordedV1['phase'], string> = {
  incoming: 'inspection.phase.incoming',
  before_operation: 'inspection.phase.beforeOperation',
  after_operation: 'inspection.phase.afterOperation',
  before_zone_closure: 'inspection.phase.beforeZoneClosure',
  assembly: 'widgets.codes.phase.assembly',
  test: 'widgets.codes.phase.test',
  final: 'inspection.phase.final',
  other: 'widgets.codes.phase.other',
}

/** Исход контроля → текст (FR-36): три разных исхода. */
export const OUTCOME_TEXT: Record<InspectionResultRecordedV1['outcome'], string> = {
  defect_indicated: 'inspection.outcome.defectFound',
  no_defect_indicated: 'inspection.outcome.noDefectFound',
  unable_to_assess: 'inspection.outcome.unableToAssess',
}

/** Пояснение к исходу: «это сигнал, а не несоответствие», «не годно и не брак». */
export const OUTCOME_NOTE: Record<InspectionResultRecordedV1['outcome'], string> = {
  defect_indicated: 'inspection.outcomeNote.defectFound',
  no_defect_indicated: 'inspection.outcomeNote.noDefectFound',
  unable_to_assess: 'inspection.outcomeNote.unableToAssess',
}

/** Почему оценка невозможна → текст. */
export const UNABLE_REASON_TEXT: Record<NonNullable<InspectionResultRecordedV1['unable_reason']>, string> = {
  poor_image: 'widgets.codes.unableReason.poorImage',
  zone_occluded: 'inspection.unableReason.zoneObscured',
  carrier_unreadable: 'inspection.unableReason.idUnreadable',
  analyzer_failure: 'inspection.unableReason.analyzerFailure',
  processing_aborted: 'inspection.unableReason.processingAborted',
  not_measured: 'widgets.codes.unableReason.notMeasured',
  document_missing: 'widgets.codes.unableReason.documentMissing',
  test_invalid: 'widgets.codes.unableReason.testInvalid',
  other: 'widgets.codes.unableReason.other',
}

/** Тяжесть → текст; `unknown` — «неизвестно», а не «малозначительный». */
export const SEVERITY_TEXT: Record<Severity, string> = {
  critical: 'inspection.severity.critical',
  major: 'inspection.severity.major',
  minor: 'inspection.severity.minor',
  unknown: 'common.words.unknown',
}

/** Рекомендация анализатора → текст (только совет, FR-48). */
export const RECOMMENDATION_TEXT: Record<NonNullable<InspectionResultRecordedV1['recommendation']>, string> = {
  pass_to_next: 'widgets.codes.recommendation.passToNext',
  manual_review: 'widgets.codes.recommendation.manualReview',
  isolate: 'widgets.codes.recommendation.isolate',
  none: 'widgets.codes.recommendation.none',
}

/** Реакция карты реакций → текст. */
export const REACTION_TEXT: Record<ReactionOutcome, string> = {
  pass_to_next: 'widgets.codes.reaction.passToNext',
  manual_review: 'widgets.codes.reaction.manualReview',
  isolate: 'widgets.codes.reaction.isolate',
  question_to_technologist: 'widgets.codes.reaction.questionToTechnologist',
}

/** Сравнение с состоянием зоны до операции → текст. */
export const COMPARISON_TEXT: Record<NonNullable<InspectionResultRecordedV1['comparison_before']>, string> = {
  was_before: 'widgets.codes.comparison.wasBefore',
  appeared: 'widgets.codes.comparison.appeared',
  changed: 'widgets.codes.comparison.changed',
  unchanged: 'widgets.codes.comparison.unchanged',
  not_comparable: 'widgets.codes.comparison.notComparable',
}

/** Статус «по изделию» → текст. */
export const NC_BY_ITEM_TEXT: Record<NcByItemStatus, string> = {
  draft: 'statuses.ncByItem.draft',
  isolated: 'statuses.ncByItem.isolated',
  awaiting_disposition: 'statuses.ncByItem.awaitingDisposition',
  awaiting_approvals: 'statuses.ncByItem.awaitingApprovals',
  in_execution: 'statuses.ncByItem.inExecution',
  awaiting_reinspection: 'statuses.ncByItem.awaitingReinspection',
  execution_verified: 'statuses.ncByItem.executionVerified',
  closed: 'statuses.ncByItem.closed',
  overdue: 'statuses.ncByItem.overdue',
}

/** Статус сигнала → текст. */
export const SIGNAL_STATUS_TEXT: Record<SignalStatus, string> = {
  under_review: 'statuses.signal.underReview',
  confirmed: 'statuses.signal.confirmed',
  rejected: 'statuses.signal.rejected',
  recheck_assigned: 'statuses.signal.recheckAssigned',
}

/** Вариант решения → название статуса (`statuses.disposition.*`). */
export const DISPOSITION_STATUS_TEXT: Record<Disposition, string> = {
  rework: 'statuses.disposition.rework',
  repair: 'statuses.disposition.repair',
  use_as_is: 'statuses.disposition.useAsIs',
  scrap: 'statuses.disposition.scrap',
  return_to_supplier: 'statuses.disposition.returnToSupplier',
}

/**
 * Текст кода по таблице или UNKNOWN(код) — неизвестный код не подменяется.
 * @param table — таблица «код → ключ»
 * @param code — код контракта
 * @param t — функция текстов
 */
export function codeText(table: Readonly<Record<string, string>>, code: string | null | undefined, t: (key: string) => string): string {
  if (code == null) return t('common.words.unknown')
  const key = table[code]
  return key ? t(key) : `UNKNOWN(${code})`
}

/** Доля в базисных пунктах → число 0…1 для формата `decimal2` (уверенность 0,87 — не 87 %). */
export const bpToFraction = (bp: number): number => bp / 10_000

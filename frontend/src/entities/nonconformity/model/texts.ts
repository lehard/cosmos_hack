/**
 * Коды контракта → ключи текстов интерфейса (NFR-UI-3, NFR-UI-4). Каждый код
 * получает свой текст; код вне таблицы показывается как UNKNOWN(код), а не
 * подменяется похожим (`codeText`). Где в ru.json процессной сессии нет
 * точного текста для кода контракта, текст взят из ru.shell.json (раздел
 * `widgets.codes`), а не из соседнего по смыслу ключа: «капиллярный» контроль
 * не называется «неразрушающим вообще».
 */
import type { DecisionQueueRowKind, NCCardStatus, NCConclusionVersionOutcome, NCSourceSignalBasisKind } from '@/shared/api/generated/model'
import type { Disposition, InspectionMethod, Severity } from './types'

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

/** Тяжесть → текст; `unknown` — «неизвестно», а не «малозначительный». */
export const SEVERITY_TEXT: Record<Severity, string> = {
  critical: 'inspection.severity.critical',
  major: 'inspection.severity.major',
  minor: 'inspection.severity.minor',
  unknown: 'common.words.unknown',
}

/** Вывод системы (реакция карты реакций) → текст — это предложение, а не решение. */
export const OUTCOME_TEXT: Record<NCConclusionVersionOutcome, string> = {
  pass_to_next: 'widgets.codes.reaction.passToNext',
  manual_review: 'widgets.codes.reaction.manualReview',
  isolate: 'widgets.codes.reaction.isolate',
  question_to_technologist: 'widgets.codes.reaction.questionToTechnologist',
}

/** На чём основан сигнал → текст. */
export const BASIS_KIND_TEXT: Record<NCSourceSignalBasisKind, string> = {
  inspection_result: 'widgets.codes.basisKind.inspectionResult',
  equipment_deviation: 'widgets.codes.basisKind.equipmentDeviation',
  check_skipped: 'widgets.codes.basisKind.checkSkipped',
  damage_on_receipt: 'widgets.codes.basisKind.damageOnReceipt',
  leak: 'widgets.codes.basisKind.leak',
  special_process_violation: 'widgets.codes.basisKind.specialProcessViolation',
  operator_report: 'widgets.codes.basisKind.operatorReport',
}

/** Статус несоответствия «по изделию» → текст. */
export const NC_STATUS_TEXT: Record<NCCardStatus, string> = {
  draft: 'statuses.ncByItem.draft',
  confirmed: 'widgets.ncCard.status.confirmed',
  disposition_set: 'widgets.ncCard.status.dispositionSet',
  verified: 'statuses.ncByItem.executionVerified',
  closed: 'statuses.ncByItem.closed',
}

/** Вариант решения → название статуса (`statuses.disposition.*`). */
export const DISPOSITION_STATUS_TEXT: Record<Disposition, string> = {
  rework: 'statuses.disposition.rework',
  repair: 'statuses.disposition.repair',
  use_as_is: 'statuses.disposition.useAsIs',
  scrap: 'statuses.disposition.scrap',
  return_to_supplier: 'statuses.disposition.returnToSupplier',
}

/** Вид строки очереди → текст. */
export const QUEUE_KIND_TEXT: Record<DecisionQueueRowKind, string> = {
  presentation: 'common.words.presentationPoint',
  signal: 'widgets.decisionQueue.kind.signal',
  isolated: 'widgets.decisionQueue.kind.isolated',
  review: 'widgets.decisionQueue.kind.review',
}

export { codeText } from '@/entities/item'

/** Доля в базисных пунктах → число 0…1 для формата `decimal2` (уверенность 0,87 — не 87 %). */
export const bpToFraction = (bp: number): number => bp / 10_000

/** Статус системного расследования в карточке (FR-51) → текст словаря `statuses.ncInvestigation`. */
export const INVESTIGATION_TEXT: Record<'none' | 'open' | 'closed', string> = {
  none: 'statuses.ncInvestigation.notStarted',
  open: 'statuses.ncInvestigation.inProgress',
  closed: 'statuses.ncInvestigation.closed',
}

/** Вид строки очереди → группа-задача с глаголом (UI-25): что от контролёра нужно. */
export const QUEUE_GROUP_TEXT: Record<DecisionQueueRowKind, string> = {
  review: 'widgets.decisionQueue.group.review',
  signal: 'widgets.decisionQueue.group.signal',
  isolated: 'widgets.decisionQueue.group.isolated',
  presentation: 'widgets.decisionQueue.group.presentation',
}

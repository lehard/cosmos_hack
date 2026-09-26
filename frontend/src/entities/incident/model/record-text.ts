/**
 * Подпись записи журнала на экранах разбора (FR-58, FR-153, NFR-UI-3/4): тип
 * записи каталога + уточнение → ключ текста vue-i18n и смысловой тон отметки.
 *
 * Тон берётся только из смысла контракта: исход контроля `defect_indicated` —
 * признак дефекта, `equipment.deviation.detected` — отклонение оборудования,
 * действие исполнителя — обстоятельство, а не вина. Тип, которого нет в
 * таблице, не подменяется похожим — показывается как UNKNOWN(тип).
 */
import { statusAxes, statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import type { JournalRecordRef } from './types'

/**
 * Смысловой тон отметки на дорожке:
 * - `normal` — контроль без признаков (нижняя граница окна);
 * - `finding` — признак дефекта (верхняя граница окна);
 * - `unable` — оценка невозможна (≠ «годно»);
 * - `deviation` — отклонение оборудования или обход блокировки;
 * - `action` — действие исполнителя или ход операции;
 * - `unknown` — тип вне таблицы.
 */
export type RecordTone = 'normal' | 'finding' | 'unable' | 'deviation' | 'action' | 'unknown'

/** Подпись записи: ключ текста (null — неизвестный тип) и параметры. */
export interface RecordText {
  key: string | null
  params: Record<string, string | number>
  tone: RecordTone
  /** Исходный тип — для UNKNOWN(…). */
  eventType: string
}

type Rule = { key: string; tone: RecordTone }

/** Уточнение → правило; `*` — для любого уточнения. */
const TABLE: Record<string, Record<string, Rule>> = {
  'inspection.result.recorded': {
    no_defect_indicated: { key: 'timeline.inspection.resultNoDefectFound', tone: 'normal' },
    defect_indicated: { key: 'timeline.inspection.resultDefectFound', tone: 'finding' },
    unable_to_assess: { key: 'timeline.inspection.resultUnableToAssess', tone: 'unable' },
  },
  'quality.signal.raised': { '*': { key: 'timeline.quality.signalReceived', tone: 'finding' } },
  'quality.inspection.missing': { '*': { key: 'timeline.quality.inspectionMissing', tone: 'unable' } },
  'operation.run.started': { '*': { key: 'timeline.operation.runStarted', tone: 'action' } },
  'operation.run.paused': { '*': { key: 'timeline.operation.runPaused', tone: 'action' } },
  'operation.run.resumed': { '*': { key: 'timeline.operation.runResumed', tone: 'action' } },
  'operation.run.finished': { '*': { key: 'timeline.operation.runFinished', tone: 'action' } },
  'operation.movement.sent': { '*': { key: 'timeline.operation.movementSent', tone: 'action' } },
  'operation.movement.received': { '*': { key: 'timeline.operation.movementReceived', tone: 'action' } },
  'operator.step.confirmed': { '*': { key: 'timeline.operator.stepConfirmed', tone: 'action' } },
  'operator.mode.changed': { '*': { key: 'timeline.operator.modeChange', tone: 'action' } },
  'operator.check.skipped': { '*': { key: 'timeline.operator.checkSkipped', tone: 'deviation' } },
  'operator.override.performed': { '*': { key: 'timeline.operator.manualIntervention', tone: 'deviation' } },
  'operator.deviation.reported': { '*': { key: 'timeline.operator.deviationReported', tone: 'action' } },
  'operator.inspection.requested': { '*': { key: 'timeline.operator.inspectionRequested', tone: 'action' } },
  // OperatorVision — только гипотеза о действии, текст это говорит.
  'operator.action.observed': { '*': { key: 'timeline.operator.actionDetected', tone: 'action' } },
  'equipment.state.changed': {
    cycle_started: { key: 'timeline.equipment.cycleStarted', tone: 'action' },
    cycle_finished: { key: 'timeline.equipment.cycleFinished', tone: 'action' },
    warning: { key: 'timeline.equipment.warning', tone: 'deviation' },
    fault: { key: 'timeline.equipment.alarm', tone: 'deviation' },
    '*': { key: 'timeline.equipment.stateChanged', tone: 'action' },
  },
  // FR-147: четвёртый слой — отклонения.
  'equipment.deviation.detected': {
    out_of_setpoint: { key: 'timeline.equipment.parameterOutOfRange', tone: 'deviation' },
    manual_override: { key: 'timeline.equipment.manualOverride', tone: 'deviation' },
    tool_life_warning: { key: 'timeline.equipment.toolLife', tone: 'deviation' },
    unplanned_program_change: { key: 'timeline.equipment.programChanged', tone: 'deviation' },
    alarm: { key: 'timeline.equipment.alarm', tone: 'deviation' },
    '*': { key: 'timeline.equipment.warning', tone: 'deviation' },
  },
  'equipment.program.changed': { '*': { key: 'timeline.equipment.programChanged', tone: 'action' } },
  'equipment.tool.changed': { '*': { key: 'timeline.equipment.toolLife', tone: 'action' } },
  'equipment.cycle.summarized': { '*': { key: 'timeline.equipment.parametersRecorded', tone: 'action' } },
}

/**
 * Подпись записи журнала.
 * @param r — запись с типом, уточнением и параметрами
 */
export function describeRecord(r: JournalRecordRef): RecordText {
  const byVariant = TABLE[r.event_type]
  const rule = byVariant ? (byVariant[r.variant ?? ''] ?? byVariant['*']) : undefined
  return rule
    ? { key: rule.key, params: r.params ?? {}, tone: rule.tone, eventType: r.event_type }
    : { key: null, params: {}, tone: 'unknown', eventType: r.variant ? `${r.event_type}/${r.variant}` : r.event_type }
}

/** Тон словаря статусов для отметки — цвета только из контракта (AD-30, NFR-UI-2). */
export const RECORD_TONE: Record<RecordTone, StatusTone> = {
  normal: 'success',
  finding: statusAxes.quality.values.signal.tone,
  unable: statusAxes.quality.values.unable_to_assess.tone,
  deviation: 'attention',
  action: 'info',
  unknown: 'neutral',
}

/** Цвет отметки. */
export const recordColor = (tone: RecordTone): string => statusPalette[RECORD_TONE[tone]]

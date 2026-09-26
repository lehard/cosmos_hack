/**
 * Правила экрана «Разбор обстоятельств» над данными проекции (FR-58, FR-153):
 * этапы «до / во время / после операции», входной дефект, окно, связанные
 * записи и состояние экрана. Чистые функции — проверяются юнит-тестами.
 */
import type { WidgetDataState } from '@/shared/config/widget'
import { describeRecord } from './record-text'
import { toMs } from './timescale'
import type { CausalWindow, CircumstanceRecord, CircumstancesModel, OperationSpan } from './types'

/** Этап относительно операции. */
export type Phase = 'before' | 'during' | 'after'

/** Этапы по порядку. */
export const PHASES: readonly Phase[] = ['before', 'during', 'after']

/**
 * Этап момента относительно операции; null — операции нет.
 * Незавершённая операция длится до бесконечности.
 */
export function phaseOf(t: string, op: OperationSpan | null): Phase | null {
  if (!op) return null
  const v = toMs(t)
  if (v < toMs(op.started_at)) return 'before'
  if (op.finished_at === null || v <= toMs(op.finished_at)) return 'during'
  return 'after'
}

/**
 * Значимое по этапам: результаты контроля изделия и отклонения — то, из чего
 * складывается «до операции чисто → во время две нештатности → после дефект»
 * (FR-149). Ход операции и рутинные действия сюда не входят.
 */
export function phaseSummary(model: CircumstancesModel): Record<Phase, CircumstanceRecord[]> {
  const out: Record<Phase, CircumstanceRecord[]> = { before: [], during: [], after: [] }
  for (const r of sortByTime(model.records)) {
    const tone = describeRecord(r).tone
    if (tone === 'action' || tone === 'unknown') continue
    const phase = phaseOf(r.occurred_at, model.operation)
    if (phase) out[phase].push(r)
  }
  return out
}

/**
 * Входной дефект: первая находка раньше начала операции. Тогда действия
 * исполнителя операции с дефектом не связываются (FR-58, проверка PRD).
 */
export function isIncomingDefect(model: CircumstancesModel): boolean {
  if (!model.window || !model.operation) return false
  return toMs(model.window.end) < toMs(model.operation.started_at)
}

/** Попадает ли запись (или её интервал) в окно возможного возникновения. */
export function inWindow(r: CircumstanceRecord, w: CausalWindow | null): boolean {
  if (!w) return false
  const a = toMs(r.occurred_at)
  const b = r.ended_at ? toMs(r.ended_at) : a
  return b >= toMs(w.start) && a <= toMs(w.end)
}

/**
 * Записи, подсвечиваемые при выборе записи: она сама, её связанные и те,
 * кто ссылается на неё.
 */
export function linkedIds(records: readonly CircumstanceRecord[], eventId: string | null): Set<string> {
  const out = new Set<string>()
  if (!eventId) return out
  out.add(eventId)
  for (const r of records) {
    if (r.event_id === eventId) r.related_event_ids?.forEach((id) => out.add(id))
    else if (r.related_event_ids?.includes(eventId)) out.add(r.event_id)
  }
  return out
}

/** Записи по времени возникновения (при равенстве — по `event_id`, AD-5). */
export const sortByTime = <T extends { occurred_at: string; event_id: string }>(rs: readonly T[]): T[] =>
  [...rs].sort((a, b) => toMs(a.occurred_at) - toMs(b.occurred_at) || a.event_id.localeCompare(b.event_id))

/** Противоречия во входных данных — экран показывает «ошибку входа», а не догадку. */
export function circumstancesIssues(model: CircumstancesModel): string[] {
  const issues: string[] = []
  for (const r of model.records) if (!Number.isFinite(toMs(r.occurred_at))) issues.push(`time:${r.event_id}`)
  if (model.window && !(toMs(model.window.start) <= toMs(model.window.end))) issues.push('window')
  const op = model.operation
  if (op && op.finished_at !== null && !(toMs(op.started_at) <= toMs(op.finished_at))) issues.push('operation')
  return issues
}

/**
 * Состояние экрана (AD-21, NFR-UI-4): ошибка входа — данные противоречивы;
 * «оценка невозможна» — окно определить нельзя; «признак дефекта» — на дорожке
 * изделия есть находка; иначе норма.
 */
export function circumstancesState(model: CircumstancesModel): WidgetDataState {
  if (circumstancesIssues(model).length) return 'input_error'
  if (!model.window) return 'unable_to_assess'
  if (model.records.some((r) => describeRecord(r).tone === 'finding')) return 'defect_indication'
  return 'normal'
}

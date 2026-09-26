/**
 * Область риска — «тающая область» (FR-61, FR-62): проверка версий, сокращение,
 * состояние. Сужение — только с основанием; изделие в области — «подвергалось
 * условиям, способным вызвать дефект», а не брак.
 */
import type { WidgetDataState } from '@/shared/config/widget'
import { SCOPE_LOCATIONS, type RiskScopeModel, type ScopeBreakdown, type ScopeVersion } from './types'

/** Сумма разбивки. */
export const breakdownTotal = (b: ScopeBreakdown): number => SCOPE_LOCATIONS.reduce((s, k) => s + b[k], 0)

/** Есть ли у правки основание: текст причины или доказательства. */
export const hasBasis = (v: ScopeVersion): boolean => Boolean(v.reason?.text.trim()) || v.evidence_event_ids.length > 0

/** Вид нарушения в версиях области. */
export type ScopeIssue =
  | { kind: 'basis_missing'; version: number }
  | { kind: 'breakdown_mismatch'; version: number }
  | { kind: 'version_order'; version: number }
  | { kind: 'narrowed_grew'; version: number }

/**
 * Нарушения: сужение без основания (FR-61: ни одно изделие не выходит из
 * области без записи основания), разбивка не сходится с размером, версии не по
 * порядку, «сужение», после которого область выросла.
 */
export function scopeIssues(model: RiskScopeModel): ScopeIssue[] {
  const out: ScopeIssue[] = []
  model.versions.forEach((v, i) => {
    const prev = model.versions[i - 1]
    if (v.change === 'narrowed' && !hasBasis(v)) out.push({ kind: 'basis_missing', version: v.scope_version })
    if (breakdownTotal(v.breakdown) !== v.size) out.push({ kind: 'breakdown_mismatch', version: v.scope_version })
    if (prev && v.scope_version <= prev.scope_version) out.push({ kind: 'version_order', version: v.scope_version })
    if (prev && v.change === 'narrowed' && v.size > prev.size) out.push({ kind: 'narrowed_grew', version: v.scope_version })
  })
  return out
}

/** Сокращение области от первой версии до текущей; null — версий нет. */
export function scopeReduction(model: RiskScopeModel): { from: number; to: number; ratio: number } | null {
  const first = model.versions[0]
  const last = model.versions[model.versions.length - 1]
  if (!first || !last) return null
  return { from: first.size, to: last.size, ratio: first.size > 0 ? (first.size - last.size) / first.size : 0 }
}

/** Текущая (последняя) версия. */
export const currentVersion = (model: RiskScopeModel): ScopeVersion | undefined => model.versions[model.versions.length - 1]

/**
 * Состояние: ошибка входа — нарушения версий; «признак дефекта» — у изделия
 * подтверждено; «оценка невозможна» — неизвестно последнее нормальное
 * состояние (нижняя граница окна); иначе норма.
 */
export function scopeState(model: RiskScopeModel): WidgetDataState {
  if (scopeIssues(model).length) return 'input_error'
  if (model.items.some((i) => i.known === 'confirmed')) return 'defect_indication'
  if (!model.last_known_good) return 'unable_to_assess'
  return 'normal'
}

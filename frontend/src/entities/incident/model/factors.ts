/**
 * Общие факторы группы несоответствий (FR-135): порядок строк и состояние.
 * Фактор, общий для всех несоответствий группы, — первым (проверка PRD).
 */
import type { WidgetDataState } from '@/shared/config/widget'
import type { CommonFactorRow, CommonFactorsModel, FactorKind } from './types'

/** Порядок видов факторов при равной доле (как в PRD §3a). */
export const FACTOR_ORDER: readonly FactorKind[] = ['machine', 'tool', 'fixture', 'program', 'performer', 'material_batch']

/** Вид фактора → ключ текста (ncCard.commonFactors.*). */
export const FACTOR_TEXT: Record<FactorKind, string> = {
  machine: 'ncCard.commonFactors.machine',
  tool: 'ncCard.commonFactors.tool',
  fixture: 'ncCard.commonFactors.fixture',
  program: 'ncCard.commonFactors.program',
  performer: 'ncCard.commonFactors.performer',
  material_batch: 'ncCard.commonFactors.materialBatch',
}

/** Строки: по убыванию совпадений, при равенстве — в порядке видов. */
export const sortFactors = (rows: readonly CommonFactorRow[]): CommonFactorRow[] =>
  [...rows].sort((a, b) => b.matches - a.matches || FACTOR_ORDER.indexOf(a.factor) - FACTOR_ORDER.indexOf(b.factor))

/** Фактор общий для всей группы. */
export const isCommonToAll = (row: CommonFactorRow, n: number): boolean => n > 0 && row.matches === n && row.value !== null

/** Противоречия: совпадений больше N или отрицательные числа. */
export function factorIssues(model: CommonFactorsModel): string[] {
  return model.rows
    .filter((r) => r.matches < 0 || r.matches > model.nc_count || r.distinct_values < 0)
    .map((r) => `factor:${r.factor}`)
}

/**
 * Состояние таблицы: ошибка входа — противоречия; «оценка невозможна» — ни
 * одного известного фактора; иначе норма (таблица — не признак дефекта).
 */
export function factorsState(model: CommonFactorsModel): WidgetDataState {
  if (factorIssues(model).length) return 'input_error'
  if (!model.rows.some((r) => r.value !== null)) return 'unable_to_assess'
  return 'normal'
}

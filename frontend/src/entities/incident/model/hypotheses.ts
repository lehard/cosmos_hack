/**
 * Гипотезы причины (FR-59, FR-135): порядок и состояние. Гипотеза — только
 * предположение; подтверждённая причина появляется лишь по решению человека.
 */
import type { WidgetDataState } from '@/shared/config/widget'
import type { Hypothesis, HypothesesModel, HypothesisStatus } from './types'

const STATUS_ORDER: Record<HypothesisStatus, number> = { confirmed: 0, recorded: 1, proposed_by_system: 2, rejected: 3 }

/** Подтверждённые — первыми, отклонённые — в конце; внутри — по уверенности вывода. */
export const sortHypotheses = (hs: readonly Hypothesis[]): Hypothesis[] =>
  [...hs].sort((a, b) => STATUS_ORDER[a.status] - STATUS_ORDER[b.status] || (b.confidence_bp ?? -1) - (a.confidence_bp ?? -1))

/** Можно ли решать по гипотезе: подтверждённую и отклонённую второй раз не решают. */
export const isOpen = (h: Hypothesis): boolean => h.status === 'proposed_by_system' || h.status === 'recorded'

/** Противоречия: уверенность вне 0…10000 б. п. */
export const hypothesisIssues = (model: HypothesesModel): string[] =>
  model.hypotheses
    .filter((h) => h.confidence_bp != null && (h.confidence_bp < 0 || h.confidence_bp > 10_000))
    .map((h) => `confidence:${h.hypothesis_id}`)

/**
 * Состояние: ошибка входа — противоречия; «оценка невозможна» — вывод не
 * категоричен и сведений не хватает (FR-58: категоричного вывода нет); иначе норма.
 */
export function hypothesesState(model: HypothesesModel): WidgetDataState {
  if (hypothesisIssues(model).length) return 'input_error'
  if (!model.conclusion_is_categorical && model.missing_information.length) return 'unable_to_assess'
  return 'normal'
}

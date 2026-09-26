/**
 * Источник данных виджета «hypothesis»: гипотезы и похожие случаи по
 * несоответствию в разборе (операция `analysis.hypothesis.list`, FR-59, FR-60)
 * и инцидент, в котором подтверждается причина.
 */
import { useFocusedIncident, useFocusedNc, useHypotheses } from '@/entities/incident'

/** Данные виджета «hypothesis». */
export function useHypothesesSource() {
  const ncId = useFocusedNc()
  const { id: incidentId } = useFocusedIncident()
  return { ncId, incidentId, ...useHypotheses(ncId) }
}

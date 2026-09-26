/**
 * Источник данных виджета «risk-scope»: область риска инцидента в разборе
 * (операции `analysis.incident.list`, `analysis.risk_scope.read`, FR-61, FR-62).
 */
import { useFocusedIncident, useRiskScope } from '@/entities/incident'

/** Данные виджета «risk-scope». */
export function useRiskScopeSource() {
  const { id: incidentId, list: incidents } = useFocusedIncident()
  return { incidentId, incidents, ...useRiskScope(incidentId) }
}

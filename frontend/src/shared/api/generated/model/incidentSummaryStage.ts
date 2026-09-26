/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Стадия расследования: словарь investigation_stage (contracts/statuses.yaml).
 */
export type IncidentSummaryStage = typeof IncidentSummaryStage[keyof typeof IncidentSummaryStage];


export const IncidentSummaryStage = {
  scope_defined: 'scope_defined',
  hypothesis: 'hypothesis',
  cause_confirmed: 'cause_confirmed',
  action_assigned: 'action_assigned',
  effectiveness_check: 'effectiveness_check',
  closed: 'closed',
} as const;

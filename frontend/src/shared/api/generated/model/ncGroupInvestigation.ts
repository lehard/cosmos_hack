/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type NcGroupInvestigation = typeof NcGroupInvestigation[keyof typeof NcGroupInvestigation];


export const NcGroupInvestigation = {
  not_required: 'not_required',
  not_started: 'not_started',
  in_progress: 'in_progress',
  hypothesis_only: 'hypothesis_only',
  cause_confirmed: 'cause_confirmed',
  cause_not_established: 'cause_not_established',
  measures_assigned: 'measures_assigned',
  effectiveness_check: 'effectiveness_check',
  closed: 'closed',
} as const;

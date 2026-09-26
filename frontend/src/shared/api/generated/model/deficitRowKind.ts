/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type DeficitRowKind = typeof DeficitRowKind[keyof typeof DeficitRowKind];


export const DeficitRowKind = {
  tool_unknown: 'tool_unknown',
  cycle_end_time_unknown: 'cycle_end_time_unknown',
  no_observation_after_operation: 'no_observation_after_operation',
  no_observation_before_operation: 'no_observation_before_operation',
  operator_unknown: 'operator_unknown',
  equipment_log_missing: 'equipment_log_missing',
  other: 'other',
} as const;

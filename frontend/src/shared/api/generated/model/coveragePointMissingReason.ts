/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type CoveragePointMissingReason = typeof CoveragePointMissingReason[keyof typeof CoveragePointMissingReason];


export const CoveragePointMissingReason = {
  result_not_received: 'result_not_received',
  check_skipped: 'check_skipped',
  point_manual_mode: 'point_manual_mode',
  not_covered_by_method: 'not_covered_by_method',
  unknown: 'unknown',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type AnalyzerCheckCheckKind = typeof AnalyzerCheckCheckKind[keyof typeof AnalyzerCheckCheckKind];


export const AnalyzerCheckCheckKind = {
  exam: 'exam',
  reference_set: 'reference_set',
  shadow_comparison: 'shadow_comparison',
  drift_monitor: 'drift_monitor',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type AnalyzerSuspensionTrigger = typeof AnalyzerSuspensionTrigger[keyof typeof AnalyzerSuspensionTrigger];


export const AnalyzerSuspensionTrigger = {
  drift: 'drift',
  reference_set_failed: 'reference_set_failed',
  disagreement_growth: 'disagreement_growth',
  escape_detected: 'escape_detected',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Визуальный контроль или контроль действий оператора.
 */
export type AnalyzerSummaryKind = typeof AnalyzerSummaryKind[keyof typeof AnalyzerSummaryKind];


export const AnalyzerSummaryKind = {
  visionqc: 'visionqc',
  operatorvision: 'operatorvision',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Стадия допуска: тень, пилот, работа.
 */
export type AnalyzerSummaryStage = typeof AnalyzerSummaryStage[keyof typeof AnalyzerSummaryStage];


export const AnalyzerSummaryStage = {
  shadow: 'shadow',
  pilot: 'pilot',
  active: 'active',
} as const;

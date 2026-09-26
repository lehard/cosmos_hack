/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type AnalyzerSummaryStatus = typeof AnalyzerSummaryStatus[keyof typeof AnalyzerSummaryStatus];


export const AnalyzerSummaryStatus = {
  active: 'active',
  suspended: 'suspended',
  retired: 'retired',
  not_admitted: 'not_admitted',
} as const;

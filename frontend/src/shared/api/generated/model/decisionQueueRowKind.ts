/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Точка предъявления, сигнал на рассмотрение, изолированное изделие, пересмотр решения, принятого до новых данных (AD-3).
 */
export type DecisionQueueRowKind = typeof DecisionQueueRowKind[keyof typeof DecisionQueueRowKind];


export const DecisionQueueRowKind = {
  presentation: 'presentation',
  signal: 'signal',
  isolated: 'isolated',
  review: 'review',
} as const;

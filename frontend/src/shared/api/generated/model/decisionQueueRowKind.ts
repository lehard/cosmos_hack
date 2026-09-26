/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Точка предъявления, сигнал на рассмотрение, изолированное изделие.
 */
export type DecisionQueueRowKind = typeof DecisionQueueRowKind[keyof typeof DecisionQueueRowKind];


export const DecisionQueueRowKind = {
  presentation: 'presentation',
  signal: 'signal',
  isolated: 'isolated',
} as const;

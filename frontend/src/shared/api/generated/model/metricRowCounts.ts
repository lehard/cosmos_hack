/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Что считается: изделия / физические дефекты / несоответствия / выполнения операций / наблюдения (результаты контроля) / предъявления / гипотезы / время. Для долей — чья это доля.
 */
export type MetricRowCounts = typeof MetricRowCounts[keyof typeof MetricRowCounts];


export const MetricRowCounts = {
  items: 'items',
  defects: 'defects',
  nonconformities: 'nonconformities',
  operations: 'operations',
  observations: 'observations',
  presentations: 'presentations',
  hypotheses: 'hypotheses',
  time: 'time',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Графа раздельного учёта FR-87: входной брак / оборудование / исполнители / гипотезы; нет — показатель вне раздельного учёта.
 */
export type MetricRowAccount = typeof MetricRowAccount[keyof typeof MetricRowAccount];


export const MetricRowAccount = {
  incoming: 'incoming',
  equipment: 'equipment',
  performer: 'performer',
  hypotheses: 'hypotheses',
} as const;

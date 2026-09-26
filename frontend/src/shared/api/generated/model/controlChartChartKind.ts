/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Вид карты: p — доля дефектных по подгруппам; xmr — индивидуальные значения и скользящий размах.
 */
export type ControlChartChartKind = typeof ControlChartChartKind[keyof typeof ControlChartChartKind];


export const ControlChartChartKind = {
  p: 'p',
  xmr: 'xmr',
} as const;

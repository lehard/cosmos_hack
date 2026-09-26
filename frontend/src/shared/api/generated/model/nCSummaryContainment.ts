/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Сдерживание изделия.
 */
export type NCSummaryContainment = typeof NCSummaryContainment[keyof typeof NCSummaryContainment];


export const NCSummaryContainment = {
  none: 'none',
  observe: 'observe',
  additional_check: 'additional_check',
  item_hold: 'item_hold',
  lot_hold: 'lot_hold',
} as const;

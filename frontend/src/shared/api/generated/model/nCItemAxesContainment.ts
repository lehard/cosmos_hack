/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Сдерживание (nonconformity).
 */
export type NCItemAxesContainment = typeof NCItemAxesContainment[keyof typeof NCItemAxesContainment];


export const NCItemAxesContainment = {
  none: 'none',
  observe: 'observe',
  additional_check: 'additional_check',
  item_hold: 'item_hold',
  lot_hold: 'lot_hold',
} as const;

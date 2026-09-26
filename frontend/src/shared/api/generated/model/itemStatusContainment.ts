/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Сдерживание (nonconformity); снятие блока ≠ годность.
 */
export type ItemStatusContainment = typeof ItemStatusContainment[keyof typeof ItemStatusContainment];


export const ItemStatusContainment = {
  none: 'none',
  observe: 'observe',
  additional_check: 'additional_check',
  item_hold: 'item_hold',
  lot_hold: 'lot_hold',
} as const;

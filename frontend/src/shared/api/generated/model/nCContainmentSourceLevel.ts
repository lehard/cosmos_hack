/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type NCContainmentSourceLevel = typeof NCContainmentSourceLevel[keyof typeof NCContainmentSourceLevel];


export const NCContainmentSourceLevel = {
  none: 'none',
  observe: 'observe',
  additional_check: 'additional_check',
  item_hold: 'item_hold',
  lot_hold: 'lot_hold',
} as const;

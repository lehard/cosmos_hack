/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type SetContainmentLevel = typeof SetContainmentLevel[keyof typeof SetContainmentLevel];


export const SetContainmentLevel = {
  none: 'none',
  observe: 'observe',
  additional_check: 'additional_check',
  item_hold: 'item_hold',
  lot_hold: 'lot_hold',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Дорожка: изделие, человек, оборудование.
 */
export type CircumstanceRecordLane = typeof CircumstanceRecordLane[keyof typeof CircumstanceRecordLane];


export const CircumstanceRecordLane = {
  item: 'item',
  person: 'person',
  equipment: 'equipment',
} as const;

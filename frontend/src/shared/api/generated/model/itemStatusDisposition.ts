/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Решение по изделию (nonconformity).
 */
export type ItemStatusDisposition = typeof ItemStatusDisposition[keyof typeof ItemStatusDisposition];


export const ItemStatusDisposition = {
  none: 'none',
  rework: 'rework',
  repair: 'repair',
  use_as_is: 'use_as_is',
  scrap: 'scrap',
  return_to_supplier: 'return_to_supplier',
} as const;

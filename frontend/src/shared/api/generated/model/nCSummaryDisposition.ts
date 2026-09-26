/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type NCSummaryDisposition = typeof NCSummaryDisposition[keyof typeof NCSummaryDisposition];


export const NCSummaryDisposition = {
  none: 'none',
  rework: 'rework',
  repair: 'repair',
  use_as_is: 'use_as_is',
  scrap: 'scrap',
  return_to_supplier: 'return_to_supplier',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * disposition команды nonconformity.disposition.set.
 */
export type NCDecisionActionDisposition = typeof NCDecisionActionDisposition[keyof typeof NCDecisionActionDisposition];


export const NCDecisionActionDisposition = {
  rework: 'rework',
  repair: 'repair',
  use_as_is: 'use_as_is',
  scrap: 'scrap',
  return_to_supplier: 'return_to_supplier',
} as const;

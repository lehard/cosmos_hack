/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * resolution команды nonconformity.presentation.resolve.
 */
export type NCDecisionActionResolution = typeof NCDecisionActionResolution[keyof typeof NCDecisionActionResolution];


export const NCDecisionActionResolution = {
  accept: 'accept',
  accept_with_concession: 'accept_with_concession',
  reject: 'reject',
  insufficient_data: 'insufficient_data',
} as const;

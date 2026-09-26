/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * resolution команды nonconformity.presentation.resolve.
 */
export type NCPresentationActionResolution = typeof NCPresentationActionResolution[keyof typeof NCPresentationActionResolution];


export const NCPresentationActionResolution = {
  accept: 'accept',
  accept_with_concession: 'accept_with_concession',
  reject: 'reject',
  insufficient_data: 'insufficient_data',
} as const;

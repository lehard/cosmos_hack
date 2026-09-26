/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * цело / отвергнуто / не проверяемо.
 */
export type VerifierCheckRowStatus = typeof VerifierCheckRowStatus[keyof typeof VerifierCheckRowStatus];


export const VerifierCheckRowStatus = {
  intact: 'intact',
  rejected: 'rejected',
  unverifiable: 'unverifiable',
} as const;

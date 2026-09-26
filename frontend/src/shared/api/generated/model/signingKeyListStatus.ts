/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type SigningKeyListStatus = typeof SigningKeyListStatus[keyof typeof SigningKeyListStatus];


export const SigningKeyListStatus = {
  active: 'active',
  revoked: 'revoked',
  expired: 'expired',
  unavailable: 'unavailable',
} as const;

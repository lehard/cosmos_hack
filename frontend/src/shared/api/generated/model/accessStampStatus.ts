/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type AccessStampStatus = typeof AccessStampStatus[keyof typeof AccessStampStatus];


export const AccessStampStatus = {
  active: 'active',
  expired: 'expired',
  revoked: 'revoked',
} as const;

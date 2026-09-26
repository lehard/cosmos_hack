/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type AccessQualificationStatus = typeof AccessQualificationStatus[keyof typeof AccessQualificationStatus];


export const AccessQualificationStatus = {
  valid: 'valid',
  expiring: 'expiring',
  expired: 'expired',
  revoked: 'revoked',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ConcessionStatus = typeof ConcessionStatus[keyof typeof ConcessionStatus];


export const ConcessionStatus = {
  active: 'active',
  exhausted: 'exhausted',
  expired: 'expired',
  revoked: 'revoked',
} as const;

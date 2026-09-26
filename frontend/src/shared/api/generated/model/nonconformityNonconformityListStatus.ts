/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type NonconformityNonconformityListStatus = typeof NonconformityNonconformityListStatus[keyof typeof NonconformityNonconformityListStatus];


export const NonconformityNonconformityListStatus = {
  draft: 'draft',
  confirmed: 'confirmed',
  disposition_set: 'disposition_set',
  verified: 'verified',
  closed: 'closed',
} as const;

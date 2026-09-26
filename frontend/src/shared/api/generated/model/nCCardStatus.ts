/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type NCCardStatus = typeof NCCardStatus[keyof typeof NCCardStatus];


export const NCCardStatus = {
  draft: 'draft',
  confirmed: 'confirmed',
  disposition_set: 'disposition_set',
  verified: 'verified',
  closed: 'closed',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type NCSummaryStatus = typeof NCSummaryStatus[keyof typeof NCSummaryStatus];


export const NCSummaryStatus = {
  draft: 'draft',
  confirmed: 'confirmed',
  disposition_set: 'disposition_set',
  verified: 'verified',
  closed: 'closed',
} as const;

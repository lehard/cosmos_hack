/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type DocumentSummaryStatus = typeof DocumentSummaryStatus[keyof typeof DocumentSummaryStatus];


export const DocumentSummaryStatus = {
  requested: 'requested',
  drafted: 'drafted',
  signing: 'signing',
  route_closed: 'route_closed',
  annulled: 'annulled',
} as const;

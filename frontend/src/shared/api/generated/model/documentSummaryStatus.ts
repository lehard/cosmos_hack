/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * live — сопроводительная карта собирается из истории, версия ещё не зафиксирована; returned — подписант вернул версию с замечанием.
 */
export type DocumentSummaryStatus = typeof DocumentSummaryStatus[keyof typeof DocumentSummaryStatus];


export const DocumentSummaryStatus = {
  requested: 'requested',
  drafted: 'drafted',
  signing: 'signing',
  route_closed: 'route_closed',
  annulled: 'annulled',
  live: 'live',
  returned: 'returned',
} as const;

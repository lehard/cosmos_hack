/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Область риска: open — идёт, closed — решение по изделиям принято (расследование — stage).
 */
export type IncidentSummaryStatus = typeof IncidentSummaryStatus[keyof typeof IncidentSummaryStatus];


export const IncidentSummaryStatus = {
  open: 'open',
  closed: 'closed',
} as const;

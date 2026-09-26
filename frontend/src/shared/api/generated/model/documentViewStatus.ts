/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type DocumentViewStatus = typeof DocumentViewStatus[keyof typeof DocumentViewStatus];


export const DocumentViewStatus = {
  requested: 'requested',
  drafted: 'drafted',
  signing: 'signing',
  route_closed: 'route_closed',
  annulled: 'annulled',
} as const;

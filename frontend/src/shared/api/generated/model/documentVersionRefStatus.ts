/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type DocumentVersionRefStatus = typeof DocumentVersionRefStatus[keyof typeof DocumentVersionRefStatus];


export const DocumentVersionRefStatus = {
  drafted: 'drafted',
  signing: 'signing',
  route_closed: 'route_closed',
  annulled: 'annulled',
  returned: 'returned',
} as const;

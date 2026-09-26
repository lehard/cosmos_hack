/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ItemDocumentRefStatus = typeof ItemDocumentRefStatus[keyof typeof ItemDocumentRefStatus];


export const ItemDocumentRefStatus = {
  drafted: 'drafted',
  in_route: 'in_route',
  closed: 'closed',
  annulled: 'annulled',
} as const;

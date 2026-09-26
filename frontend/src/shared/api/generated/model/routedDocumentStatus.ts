/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Те же значения, что у документов паспорта.
 */
export type RoutedDocumentStatus = typeof RoutedDocumentStatus[keyof typeof RoutedDocumentStatus];


export const RoutedDocumentStatus = {
  drafted: 'drafted',
  in_route: 'in_route',
  closed: 'closed',
  annulled: 'annulled',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type DocumentsDocumentListState = typeof DocumentsDocumentListState[keyof typeof DocumentsDocumentListState];


export const DocumentsDocumentListState = {
  draft: 'draft',
  signing: 'signing',
  signed: 'signed',
  annulled: 'annulled',
  returned: 'returned',
  paper: 'paper',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Состояние для реестра: черновик, на подписи, подписан, аннулирован, возвращён с замечанием, на бумаге (напечатан, ждёт заверения).
 */
export type DocumentSummaryState = typeof DocumentSummaryState[keyof typeof DocumentSummaryState];


export const DocumentSummaryState = {
  draft: 'draft',
  signing: 'signing',
  signed: 'signed',
  annulled: 'annulled',
  returned: 'returned',
  paper: 'paper',
} as const;

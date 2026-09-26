/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Статус бумажного экземпляра.
 */
export type DocumentSummaryPaperStatus = typeof DocumentSummaryPaperStatus[keyof typeof DocumentSummaryPaperStatus];


export const DocumentSummaryPaperStatus = {
  printed: 'printed',
  signed: 'signed',
  destroyed: 'destroyed',
} as const;

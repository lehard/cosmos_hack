/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface PaperQRView {
  doc_digest: string;
  document_id: string;
  /** QR векторной картинкой для печатной рамки (без внешних ресурсов). */
  svg: string;
  /** ant:doc:‹id›:‹отпечаток›. */
  text: string;
}

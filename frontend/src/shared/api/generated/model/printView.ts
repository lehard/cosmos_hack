/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface PrintView {
  doc_digest: string;
  document_id: string;
  /** Полная страница для печати: отрисовка + рамка. */
  html: string;
  printed_at: string;
  qr: string;
  /** QR как SVG — рисует сервер (Д-30). */
  qr_svg: string;
  rendering_hash: string;
  /** @minimum 1 */
  version: number;
}

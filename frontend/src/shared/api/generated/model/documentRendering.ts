/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface DocumentRendering {
  document_id: string;
  /** Каноническая отрисовка; серверного PDF нет. */
  html: string;
  rendering_hash: string;
  /** @minimum 1 */
  version: number;
}

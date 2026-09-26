/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface ScanRead {
  /** @maxLength 80 */
  doc_digest?: string;
  /** @maxLength 128 */
  document_id?: string;
  /**
     * Скан PNG или JPEG в base64.
     * @maxLength 16777216
     */
  image_b64: string;
}

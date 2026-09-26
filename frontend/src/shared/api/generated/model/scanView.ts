/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface ScanView {
  doc_digest: string;
  document_id: string;
  /** QR совпал с ожидаемым документом. */
  matches: boolean;
  /** H(байты скана) — адрес в хранилище материалов (AD-23). */
  scan_address: string;
  text: string;
}

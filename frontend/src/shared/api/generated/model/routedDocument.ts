/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DocumentRouteStage } from './documentRouteStage';
import type { RoutedDocumentStatus } from './routedDocumentStatus';

export interface RoutedDocument {
  basis_seq: number;
  /** Отпечаток — входит в QR бумажного экземпляра (AD-43). */
  doc_digest: string;
  doc_type: string;
  document_id: string;
  drafted_at: string;
  route: DocumentRouteStage[];
  /** Те же значения, что у документов паспорта. */
  status: RoutedDocumentStatus;
  template_ref: string;
  title: string;
  /** @minimum 1 */
  version: number;
}

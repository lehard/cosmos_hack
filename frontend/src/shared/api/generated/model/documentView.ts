/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DocumentApprovalStage } from './documentApprovalStage';
import type { DocumentSignatureView } from './documentSignatureView';
import type { DocumentSummaryField } from './documentSummaryField';
import type { DocumentViewContent } from './documentViewContent';
import type { DocumentViewStatus } from './documentViewStatus';
import type { DrillRef } from './drillRef';

export interface DocumentView {
  basis_seq: number;
  /** Канонические данные документа (JCS без rendering_hash). */
  content: DocumentViewContent;
  /** Отпечаток документа streebog256:…; в QR печатной рамки. */
  doc_digest: string;
  /** @minimum 1 */
  doc_format_version: number;
  document_id: string;
  /** ant:doc:‹id›:‹отпечаток› — для печатной рамки. */
  qr: string;
  /** H(render(шаблон@версия, content)). */
  rendering_hash: string;
  signatures: DocumentSignatureView[];
  /** События-источники документа. */
  source_event_ids: string[];
  /** Обязательные подписи — замороженный набор (AD-43). */
  stages: DocumentApprovalStage[];
  status: DocumentViewStatus;
  subject: DrillRef;
  /** Поля сводки уровня 2 — входят в отпечаток (AD-12). */
  summary_fields: DocumentSummaryField[];
  /** Предыдущая версия документа. */
  supersedes_version?: number;
  template: string;
  title: string;
  /** @minimum 1 */
  version: number;
}

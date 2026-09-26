/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DocumentSummaryPaperStatus } from './documentSummaryPaperStatus';
import type { DocumentSummaryStatus } from './documentSummaryStatus';
import type { DrillRef } from './drillRef';

export interface DocumentSummary {
  closed_at?: string;
  doc_digest?: string;
  document_id: string;
  drafted_at?: string;
  /** Статус бумажного экземпляра. */
  paper_status?: DocumentSummaryPaperStatus;
  status: DocumentSummaryStatus;
  /** Объект документа: изделие, несоответствие, партия… */
  subject: DrillRef;
  /** Шаблон@версия из нормативного слоя. */
  template: string;
  title: string;
  /** @minimum 1 */
  version: number;
}

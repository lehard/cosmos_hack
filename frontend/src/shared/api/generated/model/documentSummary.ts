/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DocumentSummaryClass } from './documentSummaryClass';
import type { DocumentSummaryPaperStatus } from './documentSummaryPaperStatus';
import type { DocumentSummaryStatus } from './documentSummaryStatus';
import type { DrillRef } from './drillRef';

export interface DocumentSummary {
  /** Класс документа: запись, решение, требование, вход. */
  class?: DocumentSummaryClass;
  closed_at?: string;
  doc_digest?: string;
  /** Вид документа: traveler, nc_statement, nc_disposition, generic. */
  doc_type?: string;
  document_id: string;
  drafted_at?: string;
  /** Статус бумажного экземпляра. */
  paper_status?: DocumentSummaryPaperStatus;
  /** live — сопроводительная карта собирается из истории, версия ещё не зафиксирована; returned — подписант вернул версию с замечанием. */
  status: DocumentSummaryStatus;
  /** Объект документа: изделие, несоответствие, партия… */
  subject: DrillRef;
  /** Шаблон@версия из нормативного слоя. */
  template: string;
  title: string;
  /** @minimum 1 */
  version: number;
  /** Сколько версий сформировано. */
  versions?: number;
}

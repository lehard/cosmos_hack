/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DocumentSummary } from './documentSummary';

export interface DocumentList {
  /** FR-65: документов собрано из истории изделия. */
  collected_from_history?: number;
  items: DocumentSummary[];
  /** FR-65: сколько полей заполнено вручную — у документов-проекций всегда 0. */
  manual_entries?: number;
  next_cursor?: string;
}

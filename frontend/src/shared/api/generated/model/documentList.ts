/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DocumentSummary } from './documentSummary';

export interface DocumentList {
  items: DocumentSummary[];
  next_cursor?: string;
}

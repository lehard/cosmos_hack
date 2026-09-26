/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DocumentPaperMarkPaperStatus } from './documentPaperMarkPaperStatus';

export interface DocumentPaperMark {
  at: string;
  copy_no?: string;
  event_id: string;
  paper_status: DocumentPaperMarkPaperStatus;
  /** @minimum 1 */
  version: number;
}

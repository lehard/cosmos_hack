/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DocumentVersionRefStatus } from './documentVersionRefStatus';

export interface DocumentVersionRef {
  doc_digest: string;
  drafted_at: string;
  /** Сколько подписей записано над этой версией. */
  signatures: number;
  status: DocumentVersionRefStatus;
  /** @minimum 1 */
  version: number;
}

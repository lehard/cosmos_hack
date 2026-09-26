/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ErpMessageAttemptOutcome } from './erpMessageAttemptOutcome';

export interface ErpMessageAttempt {
  at: string;
  error_code?: string;
  error_message?: string;
  external_document_ref?: string;
  outcome: ErpMessageAttemptOutcome;
}

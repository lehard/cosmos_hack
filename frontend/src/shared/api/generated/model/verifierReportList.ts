/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { VerifierReportSummary } from './verifierReportSummary';

export interface VerifierReportList {
  items: VerifierReportSummary[];
  next_cursor?: string;
}

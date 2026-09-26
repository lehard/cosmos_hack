/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { VerifierReportRefVerdict } from './verifierReportRefVerdict';

export interface VerifierReportRef {
  checked_at: string;
  report_ref: string;
  verdict: VerifierReportRefVerdict;
}

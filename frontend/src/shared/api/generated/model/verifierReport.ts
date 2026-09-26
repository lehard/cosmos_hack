/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DrillRef } from './drillRef';
import type { VerifierCheckRow } from './verifierCheckRow';
import type { VerifierReportSignatureClasses } from './verifierReportSignatureClasses';
import type { VerifierReportSummary } from './verifierReportSummary';

export interface VerifierReport {
  checks: VerifierCheckRow[];
  /** Реестр бумажных решений для сверки с оригиналами (AD-9). */
  paper_decisions: DrillRef[];
  /** Число подписей по классам происхождения (personal, paper, partner, scenario, genesis, server_attested). */
  signature_classes: VerifierReportSignatureClasses;
  /** Ключ верификатора. */
  signed_by: string;
  summary: VerifierReportSummary;
  /** Прогон в режиме scenario: временные проверки — «виртуальное время». */
  virtual_time: boolean;
}

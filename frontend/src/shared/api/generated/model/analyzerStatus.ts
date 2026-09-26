/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AnalyzerStatusStatus } from './analyzerStatusStatus';

export interface AnalyzerStatus {
  at: string;
  event_id: string;
  note?: string;
  status: AnalyzerStatusStatus;
  trigger?: string;
}

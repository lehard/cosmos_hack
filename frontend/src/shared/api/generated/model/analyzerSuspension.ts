/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AnalyzerSuspensionFallback } from './analyzerSuspensionFallback';
import type { AnalyzerSuspensionTrigger } from './analyzerSuspensionTrigger';

export interface AnalyzerSuspension {
  at: string;
  event_id: string;
  fallback: AnalyzerSuspensionFallback;
  fallback_passport_id?: string;
  trigger: AnalyzerSuspensionTrigger;
}

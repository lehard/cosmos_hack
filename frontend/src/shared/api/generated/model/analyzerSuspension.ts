/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AnalyzerSuspensionFallback } from './analyzerSuspensionFallback';
import type { AnalyzerSuspensionTrigger } from './analyzerSuspensionTrigger';

export interface AnalyzerSuspension {
  at: string;
  /** event_id записей-оснований приостановки (наблюдения, пропуск брака, отчёт проверки). */
  basis?: string[];
  event_id: string;
  fallback: AnalyzerSuspensionFallback;
  fallback_passport_id?: string;
  /** Что увидело правило автоотката, по-русски. */
  note?: string;
  /** Прогон сценария, в котором приостановлен паспорт (AD-38). */
  run_id?: string;
  trigger: AnalyzerSuspensionTrigger;
}

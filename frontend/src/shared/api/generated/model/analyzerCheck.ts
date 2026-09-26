/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AnalyzerCheckCheckKind } from './analyzerCheckCheckKind';

export interface AnalyzerCheck {
  check_kind: AnalyzerCheckCheckKind;
  /**
     * @minimum 0
     * @maximum 10000
     */
  disagreement_rate_bp?: number;
  /**
     * @minimum 0
     * @maximum 10000
     */
  escape_rate_bp?: number;
  event_id: string;
  /**
     * @minimum 0
     * @maximum 10000
     */
  false_alarm_rate_bp?: number;
  occurred_at: string;
  passed: boolean;
  passport_id: string;
}

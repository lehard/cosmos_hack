/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface DecisionEscalation {
  /**
     * @minimum 0
     * @maximum 5
     */
  automation_mode: number;
  reason: string;
  rule_id?: string;
  rule_rev?: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ReactionRuleActionClass } from './reactionRuleActionClass';
import type { ReactionRuleOutcome } from './reactionRuleOutcome';
import type { ReactionRuleSeverity } from './reactionRuleSeverity';

export interface ReactionRule {
  /** Класс действия (AD-27). */
  action_class: ReactionRuleActionClass;
  approved_by?: string;
  /**
     * Режим автоматизации 1–5 (FR-50).
     * @minimum 1
     * @maximum 5
     */
  automation_mode: number;
  defect_type_code?: string;
  outcome: ReactionRuleOutcome;
  /** Владелец правила. */
  owner: string;
  rule_id: string;
  severity?: ReactionRuleSeverity;
  /**
     * Порог уверенности, б. п.
     * @minimum 0
     * @maximum 10000
     */
  threshold_bp?: number;
  title: string;
  /** Условие срабатывания по-русски. */
  trigger: string;
  valid_until?: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { IntegrationDecisionPrevious } from './integrationDecisionPrevious';
import type { IntegrationDecisionState } from './integrationDecisionState';

export interface IntegrationDecision {
  /** Кто решил (псевдоним). */
  actor?: string;
  at: string;
  previous?: IntegrationDecisionPrevious;
  reason: string;
  seq: number;
  state: IntegrationDecisionState;
}

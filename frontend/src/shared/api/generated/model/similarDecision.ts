/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { SimilarDecisionOutcome } from './similarDecisionOutcome';

export interface SimilarDecision {
  decided_at: string;
  number: string;
  outcome: SimilarDecisionOutcome;
  ref_id: string;
  summary: string;
}

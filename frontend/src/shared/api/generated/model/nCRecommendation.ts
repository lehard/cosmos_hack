/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCRecommendationOutcome } from './nCRecommendationOutcome';

export interface NCRecommendation {
  /** Рекомендуемый исход: решение на точке или исход пересмотра. */
  outcome: NCRecommendationOutcome;
  /** Почему система это предлагает — словами. */
  why: string[];
}

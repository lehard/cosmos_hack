/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { StationSuggestionOutcome } from './stationSuggestionOutcome';

export interface StationSuggestion {
  /** stop — остановить точку, release — снять остановку. */
  outcome: StationSuggestionOutcome;
  /** Почему система это предлагает — словами. */
  why: string[];
}

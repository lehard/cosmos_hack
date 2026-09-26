/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { SuggestionHistoryType } from './suggestionHistoryType';

export interface SuggestionHistory {
  actor?: string;
  at: string;
  event_id: string;
  responsible_id?: string;
  responsible_role?: string;
  text?: string;
  type: SuggestionHistoryType;
}

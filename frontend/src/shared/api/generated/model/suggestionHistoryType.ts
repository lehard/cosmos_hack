/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type SuggestionHistoryType = typeof SuggestionHistoryType[keyof typeof SuggestionHistoryType];


export const SuggestionHistoryType = {
  recorded: 'recorded',
  forwarded: 'forwarded',
  accepted: 'accepted',
  rejected: 'rejected',
} as const;

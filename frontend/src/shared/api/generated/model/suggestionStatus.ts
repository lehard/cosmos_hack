/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Новое / передано ответственному / принято в работу / отклонено. «Принято» ничего не применяет само.
 */
export type SuggestionStatus = typeof SuggestionStatus[keyof typeof SuggestionStatus];


export const SuggestionStatus = {
  new: 'new',
  forwarded: 'forwarded',
  accepted: 'accepted',
  rejected: 'rejected',
} as const;

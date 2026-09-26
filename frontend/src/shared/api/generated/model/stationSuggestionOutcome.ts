/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * stop — остановить точку, release — снять остановку.
 */
export type StationSuggestionOutcome = typeof StationSuggestionOutcome[keyof typeof StationSuggestionOutcome];


export const StationSuggestionOutcome = {
  stop: 'stop',
  release: 'release',
} as const;

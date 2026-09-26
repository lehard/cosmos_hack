/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Для какого решения.
 */
export type ConcessionKind = typeof ConcessionKind[keyof typeof ConcessionKind];


export const ConcessionKind = {
  repair: 'repair',
  use_as_is: 'use_as_is',
} as const;

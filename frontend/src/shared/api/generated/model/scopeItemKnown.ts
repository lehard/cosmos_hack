/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Ось incident словаря статусов.
 */
export type ScopeItemKnown = typeof ScopeItemKnown[keyof typeof ScopeItemKnown];


export const ScopeItemKnown = {
  confirmed: 'confirmed',
  suspect: 'suspect',
  excluded: 'excluded',
  unknown: 'unknown',
} as const;

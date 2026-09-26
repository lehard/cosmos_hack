/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Словарь incident_action.
 */
export type ScopeItemAction = typeof ScopeItemAction[keyof typeof ScopeItemAction];


export const ScopeItemAction = {
  observe: 'observe',
  check: 'check',
  block: 'block',
  release: 'release',
} as const;

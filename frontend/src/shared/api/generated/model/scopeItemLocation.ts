/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ScopeItemLocation = typeof ScopeItemLocation[keyof typeof ScopeItemLocation];


export const ScopeItemLocation = {
  in_production: 'in_production',
  moved_on: 'moved_on',
  assembled: 'assembled',
  shipped: 'shipped',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Что делать.
 */
export type ItemIncidentAction = typeof ItemIncidentAction[keyof typeof ItemIncidentAction];


export const ItemIncidentAction = {
  observe: 'observe',
  check: 'check',
  block: 'block',
  release: 'release',
} as const;

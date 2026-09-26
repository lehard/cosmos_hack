/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Что известно.
 */
export type ItemIncidentStatus = typeof ItemIncidentStatus[keyof typeof ItemIncidentStatus];


export const ItemIncidentStatus = {
  confirmed: 'confirmed',
  suspect: 'suspect',
  excluded: 'excluded',
  unknown: 'unknown',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * not_reached — сценарий не дошёл до шага.
 */
export type BoardRowStatus = typeof BoardRowStatus[keyof typeof BoardRowStatus];


export const BoardRowStatus = {
  pending: 'pending',
  passed: 'passed',
  failed: 'failed',
  not_reached: 'not_reached',
} as const;

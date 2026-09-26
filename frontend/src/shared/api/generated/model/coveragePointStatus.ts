/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Результат получен / ещё ждём / нет (quality.inspection.missing).
 */
export type CoveragePointStatus = typeof CoveragePointStatus[keyof typeof CoveragePointStatus];


export const CoveragePointStatus = {
  received: 'received',
  pending: 'pending',
  missing: 'missing',
} as const;

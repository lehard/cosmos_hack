/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Рассмотрен ли сигнал человеком.
 */
export type QualitySignalState = typeof QualitySignalState[keyof typeof QualitySignalState];


export const QualitySignalState = {
  open: 'open',
  confirmed: 'confirmed',
  rejected: 'rejected',
} as const;

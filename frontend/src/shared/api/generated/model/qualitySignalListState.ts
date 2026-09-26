/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type QualitySignalListState = typeof QualitySignalListState[keyof typeof QualitySignalListState];


export const QualitySignalListState = {
  open: 'open',
  confirmed: 'confirmed',
  rejected: 'rejected',
} as const;

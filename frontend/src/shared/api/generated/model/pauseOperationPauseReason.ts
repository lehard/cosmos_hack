/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type PauseOperationPauseReason = typeof PauseOperationPauseReason[keyof typeof PauseOperationPauseReason];


export const PauseOperationPauseReason = {
  setup: 'setup',
  failure: 'failure',
  waiting: 'waiting',
  shift_end: 'shift_end',
  other: 'other',
  unknown: 'unknown',
} as const;

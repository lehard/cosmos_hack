/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type QuarantineEntryState = typeof QuarantineEntryState[keyof typeof QuarantineEntryState];


export const QuarantineEntryState = {
  open: 'open',
  accepted: 'accepted',
  still_invalid: 'still_invalid',
  discarded: 'discarded',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type IngestQuarantineListState = typeof IngestQuarantineListState[keyof typeof IngestQuarantineListState];


export const IngestQuarantineListState = {
  open: 'open',
  accepted: 'accepted',
  still_invalid: 'still_invalid',
  discarded: 'discarded',
} as const;

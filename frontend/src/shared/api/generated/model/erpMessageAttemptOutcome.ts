/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ErpMessageAttemptOutcome = typeof ErpMessageAttemptOutcome[keyof typeof ErpMessageAttemptOutcome];


export const ErpMessageAttemptOutcome = {
  accepted: 'accepted',
  duplicate: 'duplicate',
  rejected: 'rejected',
  transport_error: 'transport_error',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * accepted — записан; duplicate — тот же payload уже есть (повтор не меняет показатели, AD-7); quarantined — в карантине с кодом; rejected — отказ без карантина.
 */
export type IngestOutcomeStatus = typeof IngestOutcomeStatus[keyof typeof IngestOutcomeStatus];


export const IngestOutcomeStatus = {
  accepted: 'accepted',
  duplicate: 'duplicate',
  quarantined: 'quarantined',
  rejected: 'rejected',
} as const;

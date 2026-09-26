/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Операция API.
 */
export type StationActionOperation = typeof StationActionOperation[keyof typeof StationActionOperation];


export const StationActionOperation = {
  nonconformityprocess_holdset: 'nonconformity.process_hold.set',
  nonconformityprocess_holdrelease: 'nonconformity.process_hold.release',
} as const;

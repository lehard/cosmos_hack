/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ObservationAccountStatusNow = typeof ObservationAccountStatusNow[keyof typeof ObservationAccountStatusNow];


export const ObservationAccountStatusNow = {
  active: 'active',
  suspended: 'suspended',
  retired: 'retired',
  not_admitted: 'not_admitted',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Статус паспорта на момент наблюдения.
 */
export type ObservationAccountStatusThen = typeof ObservationAccountStatusThen[keyof typeof ObservationAccountStatusThen];


export const ObservationAccountStatusThen = {
  active: 'active',
  suspended: 'suspended',
  retired: 'retired',
  not_admitted: 'not_admitted',
} as const;

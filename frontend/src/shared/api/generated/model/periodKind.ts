/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type PeriodKind = typeof PeriodKind[keyof typeof PeriodKind];


export const PeriodKind = {
  shift: 'shift',
  day: 'day',
  week: 'week',
  month: 'month',
  custom: 'custom',
} as const;

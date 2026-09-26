/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Строже из оценки источника и расчёта системы.
 */
export type InspectionMeasurementVerdict = typeof InspectionMeasurementVerdict[keyof typeof InspectionMeasurementVerdict];


export const InspectionMeasurementVerdict = {
  within: 'within',
  outside: 'outside',
  not_measured: 'not_measured',
} as const;

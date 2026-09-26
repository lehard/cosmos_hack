/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type InspectionMeasurementSourceVerdict = typeof InspectionMeasurementSourceVerdict[keyof typeof InspectionMeasurementSourceVerdict];


export const InspectionMeasurementSourceVerdict = {
  within: 'within',
  outside: 'outside',
  not_measured: 'not_measured',
} as const;

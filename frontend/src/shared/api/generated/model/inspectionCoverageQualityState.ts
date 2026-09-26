/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Состояние качества изделия.
 */
export type InspectionCoverageQualityState = typeof InspectionCoverageQualityState[keyof typeof InspectionCoverageQualityState];


export const InspectionCoverageQualityState = {
  not_inspected: 'not_inspected',
  conforming: 'conforming',
  accepted_with_concession: 'accepted_with_concession',
  unable_to_assess: 'unable_to_assess',
  signal: 'signal',
  nonconforming: 'nonconforming',
} as const;

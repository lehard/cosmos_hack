/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * «Оценка невозможна» ≠ «годно».
 */
export type InspectionResultOutcome = typeof InspectionResultOutcome[keyof typeof InspectionResultOutcome];


export const InspectionResultOutcome = {
  defect_indicated: 'defect_indicated',
  no_defect_indicated: 'no_defect_indicated',
  unable_to_assess: 'unable_to_assess',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * not_checked — не проверено методом, способным выявить: не годность.
 */
export type DefectTypeCoverageStatus = typeof DefectTypeCoverageStatus[keyof typeof DefectTypeCoverageStatus];


export const DefectTypeCoverageStatus = {
  defect: 'defect',
  no_defect: 'no_defect',
  unable: 'unable',
  not_checked: 'not_checked',
} as const;

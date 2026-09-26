/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * «цело» только при нуле «не проверяемо», иначе «цело с оговорками».
 */
export type VerifierReportSummaryVerdict = typeof VerifierReportSummaryVerdict[keyof typeof VerifierReportSummaryVerdict];


export const VerifierReportSummaryVerdict = {
  intact: 'intact',
  intact_with_reservations: 'intact_with_reservations',
  violated: 'violated',
} as const;

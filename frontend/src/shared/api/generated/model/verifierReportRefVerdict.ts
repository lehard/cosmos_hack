/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type VerifierReportRefVerdict = typeof VerifierReportRefVerdict[keyof typeof VerifierReportRefVerdict];


export const VerifierReportRefVerdict = {
  intact: 'intact',
  intact_with_caveats: 'intact_with_caveats',
  violated: 'violated',
  unknown: 'unknown',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Системное расследование (FR-51).
 */
export type NCSummaryInvestigationStatus = typeof NCSummaryInvestigationStatus[keyof typeof NCSummaryInvestigationStatus];


export const NCSummaryInvestigationStatus = {
  none: 'none',
  open: 'open',
  closed: 'closed',
} as const;

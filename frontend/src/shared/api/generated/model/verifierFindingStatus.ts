/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * rejected — нарушение, unverifiable — не проверяемо, note — пояснение при «цело».
 */
export type VerifierFindingStatus = typeof VerifierFindingStatus[keyof typeof VerifierFindingStatus];


export const VerifierFindingStatus = {
  rejected: 'rejected',
  unverifiable: 'unverifiable',
  note: 'note',
} as const;

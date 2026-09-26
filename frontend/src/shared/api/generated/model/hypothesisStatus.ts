/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type HypothesisStatus = typeof HypothesisStatus[keyof typeof HypothesisStatus];


export const HypothesisStatus = {
  proposed_by_system: 'proposed_by_system',
  recorded: 'recorded',
  confirmed: 'confirmed',
  rejected: 'rejected',
} as const;

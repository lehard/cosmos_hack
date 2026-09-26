/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type DecisionProposalKind = typeof DecisionProposalKind[keyof typeof DecisionProposalKind];


export const DecisionProposalKind = {
  disposition: 'disposition',
  concession: 'concession',
  presentation: 'presentation',
  other: 'other',
} as const;

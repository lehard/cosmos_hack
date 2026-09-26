/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type HypothesisCategory = typeof HypothesisCategory[keyof typeof HypothesisCategory];


export const HypothesisCategory = {
  incoming: 'incoming',
  equipment: 'equipment',
  performer: 'performer',
  handling: 'handling',
  assembly: 'assembly',
  documentation: 'documentation',
  not_established: 'not_established',
} as const;

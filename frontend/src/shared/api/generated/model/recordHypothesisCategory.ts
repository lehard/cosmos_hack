/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type RecordHypothesisCategory = typeof RecordHypothesisCategory[keyof typeof RecordHypothesisCategory];


export const RecordHypothesisCategory = {
  incoming: 'incoming',
  equipment: 'equipment',
  performer: 'performer',
  handling: 'handling',
  assembly: 'assembly',
  documentation: 'documentation',
  not_established: 'not_established',
} as const;

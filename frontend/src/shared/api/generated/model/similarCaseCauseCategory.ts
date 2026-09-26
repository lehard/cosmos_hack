/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * @nullable
 */
export type SimilarCaseCauseCategory = typeof SimilarCaseCauseCategory[keyof typeof SimilarCaseCauseCategory] | null;


export const SimilarCaseCauseCategory = {
  incoming: 'incoming',
  equipment: 'equipment',
  performer: 'performer',
  handling: 'handling',
  assembly: 'assembly',
  documentation: 'documentation',
  not_established: 'not_established',
} as const;

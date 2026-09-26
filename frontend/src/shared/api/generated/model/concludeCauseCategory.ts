/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ConcludeCauseCategory = typeof ConcludeCauseCategory[keyof typeof ConcludeCauseCategory];


export const ConcludeCauseCategory = {
  incoming: 'incoming',
  equipment: 'equipment',
  performer: 'performer',
  handling: 'handling',
  assembly: 'assembly',
  documentation: 'documentation',
  not_established: 'not_established',
} as const;

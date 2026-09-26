/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Роль назначения (assignee_role команды access.assignment.set).
 */
export type WorkplaceCandidateRole = typeof WorkplaceCandidateRole[keyof typeof WorkplaceCandidateRole];


export const WorkplaceCandidateRole = {
  performer: 'performer',
  quality_inspector: 'quality_inspector',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Сколько подписей: 1 | все | k из n.
 */
export type DocumentApprovalStageQuorum = typeof DocumentApprovalStageQuorum[keyof typeof DocumentApprovalStageQuorum];


export const DocumentApprovalStageQuorum = {
  one: 'one',
  all: 'all',
  k_of_n: 'k_of_n',
} as const;

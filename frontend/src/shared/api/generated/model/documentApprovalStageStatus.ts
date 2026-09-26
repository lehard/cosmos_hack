/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Состояние этапа.
 */
export type DocumentApprovalStageStatus = typeof DocumentApprovalStageStatus[keyof typeof DocumentApprovalStageStatus];


export const DocumentApprovalStageStatus = {
  pending: 'pending',
  in_progress: 'in_progress',
  done: 'done',
} as const;

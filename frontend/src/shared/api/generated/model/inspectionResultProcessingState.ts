/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type InspectionResultProcessingState = typeof InspectionResultProcessingState[keyof typeof InspectionResultProcessingState];


export const InspectionResultProcessingState = {
  completed: 'completed',
  aborted: 'aborted',
  failed: 'failed',
} as const;

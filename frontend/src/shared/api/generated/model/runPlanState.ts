/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type RunPlanState = typeof RunPlanState[keyof typeof RunPlanState];


export const RunPlanState = {
  running: 'running',
  paused: 'paused',
  waiting_for_decision: 'waiting_for_decision',
  completed: 'completed',
  stopped: 'stopped',
  failed: 'failed',
} as const;

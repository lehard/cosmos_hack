/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type RunState = typeof RunState[keyof typeof RunState];


export const RunState = {
  running: 'running',
  paused: 'paused',
  waiting_for_decision: 'waiting_for_decision',
  completed: 'completed',
  stopped: 'stopped',
  failed: 'failed',
} as const;

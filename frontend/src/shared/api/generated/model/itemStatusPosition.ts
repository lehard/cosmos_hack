/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Положение в процессе (process).
 */
export type ItemStatusPosition = typeof ItemStatusPosition[keyof typeof ItemStatusPosition];


export const ItemStatusPosition = {
  in_queue: 'in_queue',
  in_progress: 'in_progress',
  at_inspection: 'at_inspection',
  at_presentation_point: 'at_presentation_point',
  in_transit: 'in_transit',
  in_storage: 'in_storage',
  isolated: 'isolated',
  completed: 'completed',
} as const;

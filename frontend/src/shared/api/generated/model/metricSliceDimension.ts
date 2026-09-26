/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type MetricSliceDimension = typeof MetricSliceDimension[keyof typeof MetricSliceDimension];


export const MetricSliceDimension = {
  location: 'location',
  step: 'step',
  equipment: 'equipment',
  performer: 'performer',
  shift: 'shift',
  defect_type: 'defect_type',
  origin: 'origin',
  cause_category: 'cause_category',
} as const;

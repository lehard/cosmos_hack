/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Измерение среза; origin — откуда брак: входной / производственный или категория подтверждённой причины (входной брак, оборудование, исполнитель…).
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
} as const;

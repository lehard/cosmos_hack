/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { MetricSliceDimension } from './metricSliceDimension';
import type { MetricValue } from './metricValue';

export interface MetricSlice {
  /** Измерение среза; origin — откуда брак: входной / производственный или категория подтверждённой причины (входной брак, оборудование, исполнитель…). */
  dimension: MetricSliceDimension;
  key: string;
  label: string;
  value: MetricValue;
}

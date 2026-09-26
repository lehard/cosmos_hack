/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ControlChartPoint } from './controlChartPoint';
import type { MetricValue } from './metricValue';

export interface ControlChart {
  center: MetricValue;
  lower: MetricValue;
  metric_id: string;
  points: ControlChartPoint[];
  step_key: string;
  upper: MetricValue;
}

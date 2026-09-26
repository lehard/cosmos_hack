/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DrillRef } from './drillRef';
import type { MetricValue } from './metricValue';

export interface ControlChartPoint {
  at: string;
  /** Выход за контрольные границы или неслучайная структура. */
  out_of_control: boolean;
  ref?: DrillRef;
  value: MetricValue;
}

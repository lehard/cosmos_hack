/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { MetricRowGroup } from './metricRowGroup';
import type { MetricSlice } from './metricSlice';
import type { MetricValue } from './metricValue';

export interface MetricRow {
  /** Раздел: раздельный учёт кейса §2.4, §5.2. */
  group: MetricRowGroup;
  metric_id: string;
  slices: MetricSlice[];
  title: string;
  total: MetricValue;
  unknown: boolean;
}

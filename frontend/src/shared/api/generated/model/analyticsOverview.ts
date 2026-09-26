/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { MetricRow } from './metricRow';
import type { Period } from './period';

export interface AnalyticsOverview {
  basis_seq: number;
  items: MetricRow[];
  period: Period;
}

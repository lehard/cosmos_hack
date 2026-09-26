/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ContributionRow } from './contributionRow';
import type { MetricValue } from './metricValue';
import type { Period } from './period';

export interface MetricDrilldown {
  items: ContributionRow[];
  metric_id: string;
  next_cursor?: string;
  period: Period;
  total: MetricValue;
}

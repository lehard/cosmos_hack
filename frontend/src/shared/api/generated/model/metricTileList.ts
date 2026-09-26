/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { MetricTile } from './metricTile';
import type { Period } from './period';

export interface MetricTileList {
  items: MetricTile[];
  period: Period;
}

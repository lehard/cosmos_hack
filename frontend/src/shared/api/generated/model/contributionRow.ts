/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { MetricValue } from './metricValue';

export interface ContributionRow {
  item_id: string;
  label: string;
  slice_key: string;
  /** id исходных записей журнала (раскрытие до записей). */
  source_event_ids: string[];
  /** Происхождение: источник факта (FR-140). */
  source_kinds: string[];
  value: MetricValue;
}

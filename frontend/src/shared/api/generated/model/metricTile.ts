/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DrillRef } from './drillRef';
import type { MetricValue } from './metricValue';

export interface MetricTile {
  /** Куда провалиться (FR-7). */
  drill?: DrillRef;
  /** Идентификатор показателя: inspected_items, items_with_confirmed_nc, first_pass_yield, defects_by_type, cause_established, lead_time, waiting_time… */
  metric_id: string;
  /** Значение за предыдущий такой же период. */
  previous?: MetricValue;
  /** Название по словарю продукта («Прохождение контроля с первого раза», Д-11). */
  title: string;
  unknown: boolean;
  value: MetricValue;
}

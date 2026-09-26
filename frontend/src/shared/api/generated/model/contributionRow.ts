/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DrillRef } from './drillRef';
import type { MetricValue } from './metricValue';

export interface ContributionRow {
  /** Момент, к которому относится вклад (FR-3). */
  at?: string;
  /** Изделие; для строк вне изделия (простой оборудования) — id объекта из ref. */
  item_id: string;
  label: string;
  /** Объект строки вне изделия — куда провалиться (FR-7). */
  ref?: DrillRef;
  slice_key: string;
  /** id исходных записей журнала (раскрытие до записей). */
  source_event_ids: string[];
  /** Виды источника исходных записей (FR-140), а не виды записей: manual_entry, machine, sensor, camera, external_system, import — для фактов; manual_entry — решение человека; system — вывод системы. */
  source_kinds: string[];
  value: MetricValue;
}

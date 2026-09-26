/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { MetricRowAccount } from './metricRowAccount';
import type { MetricRowCounts } from './metricRowCounts';
import type { MetricRowGroup } from './metricRowGroup';
import type { MetricSlice } from './metricSlice';
import type { MetricValue } from './metricValue';

export interface MetricRow {
  /** Графа раздельного учёта FR-87: входной брак / оборудование / исполнители / гипотезы; нет — показатель вне раздельного учёта. */
  account?: MetricRowAccount;
  /** Что считается: изделия / физические дефекты / несоответствия / выполнения операций / наблюдения (результаты контроля) / предъявления / гипотезы / время. Для долей — чья это доля. */
  counts?: MetricRowCounts;
  /** Раздел: раздельный учёт кейса §2.4, §5.2. */
  group: MetricRowGroup;
  metric_id: string;
  slices: MetricSlice[];
  title: string;
  total: MetricValue;
  unknown: boolean;
}

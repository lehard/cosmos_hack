/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ControlChartChartKind } from './controlChartChartKind';
import type { ControlChartPoint } from './controlChartPoint';
import type { MetricValue } from './metricValue';

export interface ControlChart {
  center: MetricValue;
  /** Вид карты: p — доля дефектных по подгруппам; xmr — индивидуальные значения и скользящий размах. */
  chart_kind?: ControlChartChartKind;
  lower: MetricValue;
  metric_id: string;
  points: ControlChartPoint[];
  step_key: string;
  /** Имя узла BPMN действующей версии процесса; нет — показывать step_key. */
  step_name?: string;
  /** Название показателя карты («Доля результатов контроля с признаком дефекта», «Длительность операции»). */
  title: string;
  upper: MetricValue;
}

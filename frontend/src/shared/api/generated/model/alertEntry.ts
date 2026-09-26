/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AlertEntryKind } from './alertEntryKind';
import type { DrillRef } from './drillRef';

export interface AlertEntry {
  alert_id: string;
  /** Вид аномалии узла (queue_above_norm, wait_above_norm, downtime_over_threshold, output_spike, defect_rate_out_of_control). */
  anomaly?: string;
  at: string;
  /** Точка предъявления. */
  gate?: string;
  /** Изделие (просроченная изоляция, не перемещено в изолятор). */
  item?: string;
  /** @minimum 0 */
  items?: number;
  kind: AlertEntryKind;
  /** Узел (step_key) аномалии. */
  node?: string;
  /** Имя узла — name элемента BPMN действующей версии процесса (для подписи вместо step_key); нет — показывать node. */
  node_name?: string;
  /** @minimum 0 */
  operations?: number;
  /** @minimum 0 */
  overdue_minutes?: number;
  /** Объект тревоги (FR-7). */
  ref?: DrillRef;
  /** Эскалация: цель. */
  target?: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DecisionQueueRowKind } from './decisionQueueRowKind';
import type { DecisionQueueRowSeverity } from './decisionQueueRowSeverity';

export interface DecisionQueueRow {
  /** seq, на котором построена строка (для basis_seq команды, AD-39). */
  basis_seq: number;
  /** Срок решения; обратный отсчёт на экране. */
  due_at?: string;
  item_id: string;
  /** Номер детали для людей. */
  item_label: string;
  /** Точка предъявления, сигнал на рассмотрение, изолированное изделие. */
  kind: DecisionQueueRowKind;
  nc_id?: string;
  /** nc_id, signal_id или id предъявления. */
  object_id: string;
  overdue: boolean;
  /** Номер предъявления (повторное — уровнем выше). */
  presentation_no?: number;
  /**
     * Порядок по риску (0 — наибольший); вычисляет сервер.
     * @minimum 0
     */
  risk_rank: number;
  severity: DecisionQueueRowSeverity;
  step_key: string;
  title: string;
}

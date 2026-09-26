/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AttentionEntryKind } from './attentionEntryKind';
import type { DrillRef } from './drillRef';

export interface AttentionEntry {
  entry_id: string;
  /**
     * overdue_decision: сколько изделий стоит.
     * @minimum 0
     */
  items?: number;
  kind: AttentionEntryKind;
  /**
     * unverified_measures, temporary_measures: сколько мер.
     * @minimum 0
     */
  n?: number;
  /**
     * overdue_decision: сколько операций стоит.
     * @minimum 0
     */
  operations?: number;
  /**
     * overdue_decision: на сколько просрочено, минуты.
     * @minimum 0
     */
  overdue_minutes?: number;
  /** Куда провалиться (FR-7). */
  ref?: DrillRef;
  /** overdue_decision: что ждёт решения — изделие, несоответствие, точка предъявления (подпись). */
  target?: string;
}

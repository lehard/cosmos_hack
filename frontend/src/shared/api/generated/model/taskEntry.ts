/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DrillRef } from './drillRef';
import type { TaskEntryKind } from './taskEntryKind';
import type { TaskEntryState } from './taskEntryState';

export interface TaskEntry {
  /**
     * Псевдоним исполнителя; null — любой с ролью.
     * @nullable
     */
  assignee_id: string | null;
  assignee_role: string;
  created_at: string;
  /**
     * Срок по производственному календарю (AD-4); null — без срока.
     * @nullable
     */
  due_at: string | null;
  kind: TaskEntryKind;
  location_id?: string;
  overdue: boolean;
  /** Субъект задачи. */
  ref?: DrillRef;
  state: TaskEntryState;
  task_id: string;
  title: string;
}

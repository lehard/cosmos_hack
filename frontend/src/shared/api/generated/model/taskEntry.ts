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
  /** Изделие задачи (субъект — изделие). */
  item_id?: string;
  /** Метка изделия для людей: номер с бирки (Ф-001), DM-код или номер из id — показывать вместо item_id. */
  item_label?: string;
  kind: TaskEntryKind;
  location_id?: string;
  /** Задача процесса (kind process_step): операция API, которой исполнитель продвигает изделие (process.movement.receive, process.operation.start, process.operation.finish, process.movement.send). */
  operation_id?: string;
  overdue: boolean;
  /** Субъект задачи. */
  ref?: DrillRef;
  state: TaskEntryState;
  /** Шаг процесса (step_key BPMN), на котором стоит изделие. */
  step_key?: string;
  task_id: string;
  title: string;
}

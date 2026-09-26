/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCHandoffStatus } from './nCHandoffStatus';

export interface NCHandoff {
  /** Решение, исполнение которого передано. */
  decision_event_id: string;
  /** Псевдоним исполнителя, если известен. */
  person?: string;
  /** Роль исполнителя по политике. */
  role_id: string;
  /** Кому передано — словами в дательном падеже («мастеру участка»). */
  role_label: string;
  /** С какого момента в этом состоянии. */
  since: string;
  /** Ожидает исполнения, исполняется, исполнено. */
  status: NCHandoffStatus;
  /** Состояние словами. */
  status_label: string;
  /** Задача notifications, если решение её породило. */
  task_id?: string;
  /** Что поручено — словами. */
  task_title: string;
}

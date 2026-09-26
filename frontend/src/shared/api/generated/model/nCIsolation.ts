/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface NCIsolation {
  decision_due_at?: string;
  /** Решение «изолировать». */
  event_id: string;
  isolated_at: string;
  isolator_location_id?: string;
  /** Срок решения истёк. */
  overdue: boolean;
  /** Перемещение в изолятор подтверждено приёмкой. */
  physically_moved: boolean;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface NCToDecide {
  /** Ремонт и «как есть» — только с действующим разрешением на отклонение. */
  concession_required: boolean;
  decision_due_at?: string;
  /** id операций решений, допустимых по состоянию (права — access.permission.list). */
  decisions: string[];
}

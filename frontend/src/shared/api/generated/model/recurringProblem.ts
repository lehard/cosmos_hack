/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface RecurringProblem {
  count: number;
  defect_type: string;
  nc_ids: string[];
  step_key: string;
  /** По инцидентам этих несоответствий есть мера. */
  with_action: boolean;
}

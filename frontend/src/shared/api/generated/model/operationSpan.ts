/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface OperationSpan {
  /**
     * null — не завершена.
     * @nullable
     */
  finished_at: string | null;
  /** Название операции по описанию процесса. */
  label: string;
  operation_run_id: string;
  started_at: string;
}

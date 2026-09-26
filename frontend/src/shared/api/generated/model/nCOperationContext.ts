/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface NCOperationContext {
  equipment_id?: string;
  /** Название оборудования — из справочника оборудования; нет в справочнике — поля нет. */
  equipment_label?: string;
  finished_at?: string;
  /** Название операции по описанию процесса. */
  label: string;
  operation_run_id: string;
  /** Псевдоним исполнителя; нет — неизвестно. */
  performer_id?: string;
  program_ref?: string;
  started_at?: string;
  step_key: string;
  tool_id?: string;
}

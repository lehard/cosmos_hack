/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */

export interface Permission {
  /** x-ant-action id ‹модуль›.‹объект›.‹действие› */
  action: string;
  /** Вид объекта (EntityKind) или all */
  subject: string;
  /** Для списка по объекту — его id */
  object_id?: string;
}

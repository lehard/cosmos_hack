/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface RunWait {
  /** x-ant-action id ожидаемой операции. */
  action: string;
  /** Объект решения (несоответствие, изделие…). */
  object_id: string;
  /** Роль стола, где ждут решения. */
  role: string;
}

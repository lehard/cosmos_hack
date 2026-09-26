/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface Violation {
  /** Код нарушения. */
  code: string;
  /** Идентификатор элемента BPMN. */
  element_id?: string;
  /** JSON Pointer поля. */
  field?: string;
  /** Пояснение. */
  message?: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */

/**
 * Либо persona_id (демо), либо login (+ password — пока необязателен).
 */
export interface SessionCreate {
  persona_id?: string;
  /** @maxLength 128 */
  login?: string;
  /** @maxLength 256 */
  password?: string;
}

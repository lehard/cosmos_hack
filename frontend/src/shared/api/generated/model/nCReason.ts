/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface NCReason {
  /** Машинный код причины. */
  code?: string;
  /**
     * Текст основания по-русски.
     * @minLength 1
     * @maxLength 2000
     */
  text: string;
}

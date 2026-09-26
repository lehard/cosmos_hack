/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface SessionCreate {
  /**
     * Логин.
     * @maxLength 128
     */
  login?: string;
  /**
     * Пароль (argon2id на сервере, эпик 08).
     * @maxLength 256
     */
  password?: string;
  /**
     * Демо-персона (только профили fixtures и demo).
     * @maxLength 64
     */
  persona_id?: string;
}

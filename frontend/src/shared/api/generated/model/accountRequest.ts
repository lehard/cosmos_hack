/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface AccountRequest {
  /**
     * Отображаемое имя (условное).
     * @minLength 1
     * @maxLength 128
     */
  display_name: string;
  /**
     * Логин: латиница в нижнем регистре, цифры, «.», «_», «-».
     * @pattern ^[a-z0-9._-]{3,64}$
     */
  login: string;
  /**
     * Пароль (на сервере — только хеш argon2id).
     * @minLength 8
     * @maxLength 256
     */
  password: string;
}

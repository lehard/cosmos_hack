/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface AccessReason {
  /**
     * Машинный код основания.
     * @maxLength 64
     */
  code?: string;
  /**
     * Основание по-русски.
     * @maxLength 2000
     */
  text: string;
}

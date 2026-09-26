/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface KnownCountsView {
  /**
     * Подтверждено.
     * @minimum 0
     */
  confirmed: number;
  /**
     * Исключено с основанием.
     * @minimum 0
     */
  excluded: number;
  /**
     * Под подозрением.
     * @minimum 0
     */
  suspect: number;
  /**
     * Нет данных — исключать нельзя.
     * @minimum 0
     */
  unknown: number;
}

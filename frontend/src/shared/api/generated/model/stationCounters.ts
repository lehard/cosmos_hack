/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface StationCounters {
  /**
     * Физические дефекты за период, не наблюдения (соглашение «Дефект»).
     * @minimum 0
     */
  defects: number;
  /**
     * В работе.
     * @minimum 0
     */
  in_progress: number;
  /**
     * Открытые несоответствия шага.
     * @minimum 0
     */
  nonconformities: number;
  /**
     * Прошли шаг.
     * @minimum 0
     */
  passed: number;
  /**
     * Ждут на шаге.
     * @minimum 0
     */
  queue: number;
}

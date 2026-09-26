/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface NodeCounters {
  /** @minimum 0 */
  defects: number;
  /** @minimum 0 */
  in_progress: number;
  /**
     * Открытые несоответствия узла (FR-154).
     * @minimum 0
     */
  nonconformities?: number;
  /** @minimum 0 */
  passed: number;
  /** @minimum 0 */
  queue: number;
  step_key: string;
}

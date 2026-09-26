/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface NCParameterReading {
  /** Наблюдённый максимум (у отклонения — значение). */
  observed_max?: number;
  /** Наблюдённый минимум (у отклонения — значение). */
  observed_min?: number;
  /** Параметр у источника (current_a — ток сварки, …). */
  parameter: string;
  /**
     * Знаков после запятой: значение = число × 10^(−scale).
     * @minimum 0
     * @maximum 12
     */
  scale: number;
  /** Уставка: верхняя граница. */
  setpoint_max?: number;
  /** Уставка: нижняя граница. */
  setpoint_min?: number;
  /** Уставка: номинал. */
  setpoint_nominal?: number;
  /** Единица, код UCUM (A, V, mm, …). */
  unit: string;
}

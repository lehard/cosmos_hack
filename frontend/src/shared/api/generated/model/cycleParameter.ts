/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface CycleParameter {
  /**
     * В пределах уставки; null — оценка невозможна.
     * @nullable
     */
  in_range: boolean | null;
  parameter: string;
  scale: number;
  /** Уставка и допуск текстом с единицей. */
  setpoint?: string;
  unit: string;
  /**
     * Значение = value × 10^(−scale); null — нет данных.
     * @nullable
     */
  value: number | null;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DataGap } from './dataGap';

export interface LaneQuality {
  /** Пропуски данных. */
  gaps: DataGap[];
  /**
     * Записей, пришедших позже, чем произошли (опоздания).
     * @minimum 0
     */
  late_count: number;
  /**
     * Наибольшее опоздание, минут.
     * @minimum 0
     */
  max_delay_min: number;
}

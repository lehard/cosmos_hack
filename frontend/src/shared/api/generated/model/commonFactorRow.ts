/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { CommonFactorRowFactor } from './commonFactorRowFactor';

export interface CommonFactorRow {
  /** @minimum 0 */
  distinct_values: number;
  factor: CommonFactorRowFactor;
  /** @minimum 0 */
  matches: number;
  /**
     * Самое частое значение; null — неизвестно.
     * @nullable
     */
  value: string | null;
}

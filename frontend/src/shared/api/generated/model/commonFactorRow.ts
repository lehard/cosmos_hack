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
  /** Значение словами: оборудование — справочник оборудования, исполнитель — справочник людей, партия — справочник партий. Нет — показать value. */
  value_label?: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { LotItemRelation } from './lotItemRelation';

export interface LotItem {
  /** Сборка, в которую вошло изделие партии (FR-45). */
  assembled?: boolean;
  item_id: string;
  label: string;
  /** Связь генеалогии. */
  relation: LotItemRelation;
  /** Изделие партии, через которое сборка попала в список. */
  via?: string;
}

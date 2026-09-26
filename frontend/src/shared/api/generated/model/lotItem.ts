/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { LotItemRelation } from './lotItemRelation';

export interface LotItem {
  item_id: string;
  label: string;
  /** Связь генеалогии. */
  relation: LotItemRelation;
}

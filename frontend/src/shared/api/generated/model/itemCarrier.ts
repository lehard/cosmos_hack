/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ItemCarrierCarrierType } from './itemCarrierCarrierType';
import type { ItemCarrierState } from './itemCarrierState';

export interface ItemCarrier {
  applied_at: string;
  carrier_type: ItemCarrierCarrierType;
  removed_at?: string;
  state: ItemCarrierState;
  temporary: boolean;
  value: string;
  zone_id?: string;
}

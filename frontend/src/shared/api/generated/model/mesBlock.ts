/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { MesBlockOutcome } from './mesBlockOutcome';

export interface MesBlock {
  business_key: string;
  error_code?: string;
  /** true — заблокировать, false — снять. */
  hold: boolean;
  item_id?: string;
  lot_id?: string;
  /** Квитанция MES; пусто — ждём. */
  outcome?: MesBlockOutcome;
  requested_at: string;
  responded_at?: string;
}

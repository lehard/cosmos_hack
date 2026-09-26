/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ErpChannelState } from './erpChannelState';
import type { ErpChannelSystem } from './erpChannelSystem';

export interface ErpChannel {
  contract_version: string;
  detail?: string;
  /** Адрес (stand или реальная система). */
  endpoint: string;
  /** @nullable */
  last_exchange_at: string | null;
  /** @minimum 0 */
  quarantined: number;
  /** @minimum 0 */
  queued: number;
  /** Работает stand — эмулятор кейса. */
  stand: boolean;
  state: ErpChannelState;
  system: ErpChannelSystem;
}

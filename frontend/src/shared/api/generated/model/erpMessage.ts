/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ErpMessageAttempt } from './erpMessageAttempt';
import type { ErpMessageExternalSystem } from './erpMessageExternalSystem';
import type { ErpMessageStatus } from './erpMessageStatus';

export interface ErpMessage {
  /** Ось «учёт в 1С» после квитанции (axis_erp_accounting). */
  accounting_state?: string;
  /** Учётное действие порта учёта; return_from_defect — «возврат из брака в производство» (Д-17). */
  action: string;
  after_rework: boolean;
  attempts: ErpMessageAttempt[];
  basis_seq: number;
  business_key: string;
  external_system: ErpMessageExternalSystem;
  item_id?: string;
  lot_id?: string;
  /** @minimum 1 */
  message_version: number;
  /** Реакция erp.posting.requested. */
  request_event_id: string;
  requested_at: string;
  /** Словарь erp_message_status (не ось). */
  status: ErpMessageStatus;
}

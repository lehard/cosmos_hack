/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DrillRef } from './drillRef';
import type { ErpOrderExternalSystem } from './erpOrderExternalSystem';

export interface ErpOrder {
  /** Срок (дата). */
  due_date?: string;
  /** Номер задания во внешней системе. */
  external_number: string;
  external_system: ErpOrderExternalSystem;
  item_revision?: string;
  item_type_id: string;
  /**
     * Изделий запущено по заданию.
     * @minimum 0
     */
  launched: number;
  order_id: string;
  /** @minimum 0 */
  quantity: number;
  received_at: string;
  ref?: DrillRef;
}

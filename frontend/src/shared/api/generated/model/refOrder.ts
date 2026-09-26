/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface RefOrder {
  /** Срок YYYY-MM-DD. */
  due_date?: string;
  external_number: string;
  external_system: string;
  item_revision?: string;
  item_type_id: string;
  order_id: string;
  quantity: number;
  received_at: string;
}

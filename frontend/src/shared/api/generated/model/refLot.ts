/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface RefLot {
  certificate_no?: string;
  /** Момент окончания годности (начало следующих местных суток). */
  expires_at?: string;
  /** Срок годности YYYY-MM-DD (включительно, местная дата). */
  expiry_date?: string;
  external_number: string;
  external_system: string;
  heat_no?: string;
  item_type_id: string;
  lot_id: string;
  quantity: number;
  received_at: string;
  supplier_id: string;
  /** Срок годности не истёк на момент ответа. */
  usable: boolean;
}

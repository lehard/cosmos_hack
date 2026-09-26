/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface LotIssue {
  at: string;
  event_id: string;
  issued_by: string;
  order_id?: string;
  /** @minimum 1 */
  quantity: number;
  to_location_id?: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface StoppedItem {
  consumer: string;
  error: string;
  failed_at: string;
  failed_seq: number;
  failure_event_id: string;
  item_id: string;
  /** @minimum 0 */
  retries: number;
}

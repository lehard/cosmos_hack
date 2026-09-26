/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface ItemHistoryEntry {
  after?: string;
  author?: string;
  before?: string;
  event_id: string;
  event_type: string;
  /** Что изменилось (ось статуса, носитель, зона…). */
  field: string;
  reason?: string;
  recorded_at: string;
  seq: number;
}

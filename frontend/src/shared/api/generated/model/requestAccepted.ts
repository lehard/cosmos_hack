/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface RequestAccepted {
  command_id: string;
  doc_digest?: string;
  document_id: string;
  event_ids: string[];
  replayed: boolean;
  seq: number;
  /** Версия, которую оформит запрос; 0 — содержимое не изменилось, новой версии нет. */
  version?: number;
}

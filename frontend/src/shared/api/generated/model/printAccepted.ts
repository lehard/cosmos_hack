/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface PrintAccepted {
  command_id: string;
  doc_digest: string;
  event_ids: string[];
  /** Страница для печати (GET). */
  print_url: string;
  qr: string;
  replayed: boolean;
  seq: number;
  /**
     * Печатаемая версия (у живой карты — новая версия, зафиксированная этой печатью).
     * @minimum 1
     */
  version: number;
}

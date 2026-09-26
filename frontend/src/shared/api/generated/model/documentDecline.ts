/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface DocumentDecline {
  at: string;
  /** Замечание. */
  comment: string;
  /** Запись document.signature.declined. */
  event_id: string;
  /** Кто вернул. */
  signer_id: string;
  /** @minimum 1 */
  stage: number;
  /** @minimum 1 */
  version: number;
}

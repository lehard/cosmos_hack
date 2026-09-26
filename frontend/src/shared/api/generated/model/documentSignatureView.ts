/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DocumentSignatureViewMethod } from './documentSignatureViewMethod';
import type { DocumentSignatureViewProvenanceClass } from './documentSignatureViewProvenanceClass';
import type { DocumentSignatureViewVerification } from './documentSignatureViewVerification';

export interface DocumentSignatureView {
  /** Заверитель бумажной подписи (AD-43). */
  attested_by?: string;
  /** Подпись над текущей версией; false — «по прежней версии», не засчитывается. */
  current_version: boolean;
  /** Отпечаток, над которым поставлена подпись. */
  doc_digest: string;
  /** Запись document.signature.recorded. */
  event_id: string;
  /**
     * @minimum 0
     * @maximum 3
     */
  level: number;
  /** source_decision — этап закрыт самим решением-источником (by_source). */
  method: DocumentSignatureViewMethod;
  /** Класс происхождения подписи (AD-2). */
  provenance_class: DocumentSignatureViewProvenanceClass;
  signed_at: string;
  signer_person_id: string;
  /** @minimum 1 */
  stage: number;
  /** Статус автоматической проверки подписи (FR-68); недоступный ключ — unverifiable, никогда не valid (AD-32). */
  verification: DocumentSignatureViewVerification;
}

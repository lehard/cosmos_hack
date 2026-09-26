/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DocumentStageSignatureCheck } from './documentStageSignatureCheck';
import type { DocumentStageSignatureClass } from './documentStageSignatureClass';
import type { DocumentStageSignatureMethod } from './documentStageSignatureMethod';

export interface DocumentStageSignature {
  /** Заверитель бумажной подписи (AD-43). */
  attested_by?: string;
  /** Проверка подписи (FR-68): демо без агента токена — unchecked. */
  check: DocumentStageSignatureCheck;
  /** Класс происхождения подписи (AD-2). */
  class: DocumentStageSignatureClass;
  /** Засчитана модулем documents: текущий отпечаток, полномочие и клеймо, уровень, разделение обязанностей. */
  counted: boolean;
  /** Запись document.signature.recorded или решение-источник (by_source). */
  event_id: string;
  /**
     * Уровень подписи (AD-13).
     * @minimum 0
     * @maximum 3
     */
  level: number;
  method: DocumentStageSignatureMethod;
  /** Учётный номер бумажного оригинала в архиве ОТК. */
  paper_original_ref?: string;
  /** Подпись по прежней версии — видна, но не засчитывается. */
  previous_version?: boolean;
  /** Адрес скана в хранилище материалов. */
  scan_address?: string;
  signed_at: string;
  /** Псевдоним подписанта или id устройства. */
  signer_id: string;
  /** Почему не засчитана: prior_version, authority, no_stamp, level, separation_*, paper_forbidden, attester_*, surplus. */
  why?: string;
}

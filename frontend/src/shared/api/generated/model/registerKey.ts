/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DsseEnvelope } from './dsseEnvelope';
import type { RegisterKeyAlgorithm } from './registerKeyAlgorithm';
import type { RegisterKeyKeyStorage } from './registerKeyKeyStorage';
import type { RegisterKeyProfileId } from './registerKeyProfileId';
import type { RegisterKeyStorageVariant } from './registerKeyStorageVariant';
import type { RegisterKeySubjectConfirmation } from './registerKeySubjectConfirmation';
import type { RegisterKeySubjectKind } from './registerKeySubjectKind';

export interface RegisterKey {
  algorithm: RegisterKeyAlgorithm;
  /**
     * seq, на котором клиент видел объект (basis_seq из ответа чтения): после него в потоках гарда не должно быть новых записей guard_relevant, иначе 409 journal.stale_state (AD-39).
     * @minimum 0
     */
  basis_seq: number;
  /** UUIDv7 клиента; повтор с тем же id возвращает прежний ответ (AD-7). У подписанной команды — event_id пакета. */
  command_id: string;
  /** Документ акта с маршрутом подписей (AD-13). */
  document_id?: string;
  /** key_id@версия. */
  key_ref: string;
  /** Класс хранения ключа человека (AD-11, AD-14, Д-72): физический ключ или ключ в браузере под PIN. */
  key_storage?: RegisterKeyKeyStorage;
  /** @minItems 1 */
  payload_classes: string[];
  /**
     * Версия политики, по которой показаны права (policy_seq сеанса); изменилась — 409 journal.stale_policy (AD-39).
     * @minimum 0
     */
  policy_seq: number;
  profile_id: RegisterKeyProfileId;
  /** Подпись нового ключа над актом. */
  proof_of_possession_b64: string;
  public_key_b64: string;
  /**
     * Первичная выдача: кто заверил расписку (≠ субъект, AD-43).
     * @maxLength 64
     */
  receipt_attested_by?: string;
  /**
     * Первичная выдача: учётный номер бумажной расписки субъекта с отпечатком ключа (архив ОТК, AD-11).
     * @maxLength 64
     */
  receipt_original_no?: string;
  /** Заменяемый ключ (ротация). */
  rotates?: string;
  /** Подписанный пакет DSSE для операций с уровнем подписи ≥ 1 (AD-10, AD-13, AD-14): подписывает агент токена, сервер сверяет отпечаток. */
  signature?: DsseEnvelope;
  /** Где лежит ключ в браузере: расширение или хранилище страницы. */
  storage_variant?: RegisterKeyStorageVariant;
  subject_confirmation: RegisterKeySubjectConfirmation;
  /** @maxLength 128 */
  subject_id: string;
  subject_kind: RegisterKeySubjectKind;
  valid_from: string;
  valid_until?: string;
  /**
     * Рабочее место сеанса (барьер 2, AD-15).
     * @maxLength 128
     */
  workplace_id?: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DsseEnvelope } from './dsseEnvelope';

export interface RecordSignature {
  /**
     * seq, на котором клиент видел объект (basis_seq из ответа чтения): после него в потоках гарда не должно быть новых записей guard_relevant, иначе 409 journal.stale_state (AD-39).
     * @minimum 0
     */
  basis_seq: number;
  /** UUIDv7 клиента; повтор с тем же id возвращает прежний ответ (AD-7). У подписанной команды — event_id пакета. */
  command_id: string;
  /** Отпечаток, который увидел подписант; расхождение — signing.document_changed. Пусто — отпечаток версии. */
  doc_digest?: string;
  /**
     * Ключ подписанта key_id@версия (агент токена).
     * @maxLength 128
     */
  key_ref?: string;
  /**
     * Версия политики, по которой показаны права (policy_seq сеанса); изменилась — 409 journal.stale_policy (AD-39).
     * @minimum 0
     */
  policy_seq: number;
  /** Подписанный пакет DSSE для операций с уровнем подписи ≥ 1 (AD-10, AD-13, AD-14): подписывает агент токена, сервер сверяет отпечаток. */
  signature?: DsseEnvelope;
  /**
     * Подпись агента над отпечатком (base64); проверяет signing (эпик 27). Пусто — демо без агента токена.
     * @maxLength 16384
     */
  signature_b64?: string;
  /**
     * Этап маршрута.
     * @minimum 1
     */
  stage: number;
  /**
     * Подписываемая версия.
     * @minimum 1
     */
  version: number;
  /**
     * Рабочее место сеанса (барьер 2, AD-15).
     * @maxLength 128
     */
  workplace_id?: string;
}

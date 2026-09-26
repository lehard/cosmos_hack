/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DsseEnvelope } from './dsseEnvelope';

export interface RequestVersion {
  /**
     * Недоступное действие (x-ant-action), для которого запрашивается решение; по нему сервер выбирает шаблон.
     * @maxLength 128
     */
  action?: string;
  /**
     * seq, на котором клиент видел объект (basis_seq из ответа чтения): после него в потоках гарда не должно быть новых записей guard_relevant, иначе 409 journal.stale_state (AD-39).
     * @minimum 0
     */
  basis_seq: number;
  /** UUIDv7 клиента; повтор с тем же id возвращает прежний ответ (AD-7). У подписанной команды — event_id пакета. */
  command_id: string;
  /** @maxLength 2000 */
  comment?: string;
  /**
     * Решение, которое оформляется.
     * @maxLength 128
     */
  decision?: string;
  /**
     * Документ, новую версию которого запрашивают; пусто — новый документ.
     * @maxLength 128
     */
  document_id?: string;
  /**
     * Версия политики, по которой показаны права (policy_seq сеанса); изменилась — 409 journal.stale_policy (AD-39).
     * @minimum 0
     */
  policy_seq: number;
  /** Подписанный пакет DSSE для операций с уровнем подписи ≥ 1 (AD-10, AD-13, AD-14): подписывает агент токена, сервер сверяет отпечаток. */
  signature?: DsseEnvelope;
  /**
     * Записи-основания.
     * @maxItems 50
     */
  source_event_ids?: string[];
  /**
     * Объект: item:‹id›, nonconformity:‹id›, process_version:‹id›…
     * @minLength 3
     * @maxLength 160
     */
  subject_ref: string;
  /**
     * Шаблон ‹id›[@‹версия›] явно (traveler — новая версия сопроводительной карты).
     * @maxLength 128
     */
  template_ref?: string;
  /**
     * Рабочее место сеанса (барьер 2, AD-15).
     * @maxLength 128
     */
  workplace_id?: string;
}

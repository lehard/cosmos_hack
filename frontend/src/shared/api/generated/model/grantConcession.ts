/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DsseEnvelope } from './dsseEnvelope';
import type { GrantConcessionKind } from './grantConcessionKind';
import type { NCReason } from './nCReason';

export interface GrantConcession {
  /**
     * seq, на котором клиент видел объект (basis_seq из ответа чтения): после него в потоках гарда не должно быть новых записей guard_relevant, иначе 409 journal.stale_state (AD-39).
     * @minimum 0
     */
  basis_seq: number;
  /** UUIDv7 клиента; повтор с тем же id возвращает прежний ответ (AD-7). У подписанной команды — event_id пакета. */
  command_id: string;
  /**
     * Пусто — присвоит сервер.
     * @maxLength 128
     */
  concession_id?: string;
  /**
     * Документ разрешения с маршрутом подписей.
     * @maxLength 128
     */
  document_id?: string;
  kind: GrantConcessionKind;
  /**
     * Лимит количества изделий.
     * @minimum 1
     */
  limit: number;
  /** @maxLength 64 */
  number?: string;
  /**
     * Версия политики, по которой показаны права (policy_seq сеанса); изменилась — 409 journal.stale_policy (AD-39).
     * @minimum 0
     */
  policy_seq: number;
  reason: NCReason;
  /**
     * Пункт КД/ТУ.
     * @maxLength 256
     */
  requirement_ref?: string;
  /** Область действия — перечень изделий. */
  scope_item_ids?: string[];
  /** @maxLength 128 */
  scope_range_from?: string;
  /** @maxLength 128 */
  scope_range_to?: string;
  /** Подписанный пакет DSSE для операций с уровнем подписи ≥ 1 (AD-10, AD-13, AD-14): подписывает агент токена, сервер сверяет отпечаток. */
  signature?: DsseEnvelope;
  /**
     * @minLength 1
     * @maxLength 256
     */
  title: string;
  valid_until?: string;
  /**
     * Рабочее место сеанса (барьер 2, AD-15).
     * @maxLength 128
     */
  workplace_id?: string;
}

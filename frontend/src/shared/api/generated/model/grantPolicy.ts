/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DsseEnvelope } from './dsseEnvelope';
import type { GrantPolicyKind } from './grantPolicyKind';
import type { GrantPolicyLimits } from './grantPolicyLimits';

export interface GrantPolicy {
  /**
     * Для kind=authority.
     * @maxLength 64
     */
  authority_id?: string;
  /**
     * seq, на котором клиент видел объект (basis_seq из ответа чтения): после него в потоках гарда не должно быть новых записей guard_relevant, иначе 409 journal.stale_state (AD-39).
     * @minimum 0
     */
  basis_seq: number;
  /** UUIDv7 клиента; повтор с тем же id возвращает прежний ответ (AD-7). У подписанной команды — event_id пакета. */
  command_id: string;
  /**
     * Документ выдачи с маршрутом подписей (AD-13, AD-43).
     * @maxLength 128
     */
  document_id?: string;
  /**
     * Вид контроля клейма.
     * @maxLength 64
     */
  inspection_kind?: string;
  kind: GrantPolicyKind;
  /** Рамки полномочия: программы, типы изделий, тяжесть, виды решений. */
  limits?: GrantPolicyLimits;
  /**
     * Приказ (для клейма обязателен).
     * @maxLength 256
     */
  order_ref?: string;
  /** @maxLength 64 */
  person_id: string;
  /**
     * Версия политики, по которой показаны права (policy_seq сеанса); изменилась — 409 journal.stale_policy (AD-39).
     * @minimum 0
     */
  policy_seq: number;
  /**
     * Для kind=role.
     * @maxLength 64
     */
  role_id?: string;
  /** @maxLength 256 */
  scope: string;
  /** Подписанный пакет DSSE для операций с уровнем подписи ≥ 1 (AD-10, AD-13, AD-14): подписывает агент токена, сервер сверяет отпечаток. */
  signature?: DsseEnvelope;
  /**
     * Для kind=stamp.
     * @maxLength 64
     */
  stamp_id?: string;
  valid_from: string;
  valid_until?: string;
  /**
     * Рабочее место сеанса (барьер 2, AD-15).
     * @maxLength 128
     */
  workplace_id?: string;
}

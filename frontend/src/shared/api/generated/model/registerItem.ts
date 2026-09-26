/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DsseEnvelope } from './dsseEnvelope';

export interface RegisterItem {
  /**
     * seq, на котором клиент видел объект (basis_seq из ответа чтения): после него в потоках гарда не должно быть новых записей guard_relevant, иначе 409 journal.stale_state (AD-39).
     * @minimum 0
     */
  basis_seq: number;
  /** UUIDv7 клиента; повтор с тем же id возвращает прежний ответ (AD-7). У подписанной команды — event_id пакета. */
  command_id: string;
  /** @maxLength 128 */
  entry_step_key?: string;
  is_assembly?: boolean;
  /** @maxLength 64 */
  item_revision: string;
  /** @maxLength 128 */
  item_type_id: string;
  /**
     * Локальный номер изделия (например, F-031); пусто — система выдаёт сама. Из метки не выводится (AD-16).
     * @maxLength 96
     */
  local_id?: string;
  lot_ids?: string[];
  /**
     * Задание 1С (сквозной сценарий §1.5).
     * @maxLength 128
     */
  order_id?: string;
  /**
     * Версия политики, по которой показаны права (policy_seq сеанса); изменилась — 409 journal.stale_policy (AD-39).
     * @minimum 0
     */
  policy_seq: number;
  /** Подписанный пакет DSSE для операций с уровнем подписи ≥ 1 (AD-10, AD-13, AD-14): подписывает агент токена, сервер сверяет отпечаток. */
  signature?: DsseEnvelope;
  /**
     * Рабочее место сеанса (барьер 2, AD-15).
     * @maxLength 128
     */
  workplace_id?: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AdmitPassportStage } from './admitPassportStage';
import type { AdmitPassportVersions } from './admitPassportVersions';
import type { DsseEnvelope } from './dsseEnvelope';

export interface AdmitPassport {
  /** @maxLength 128 */
  analyzer_id: string;
  /**
     * seq, на котором клиент видел объект (basis_seq из ответа чтения): после него в потоках гарда не должно быть новых записей guard_relevant, иначе 409 journal.stale_state (AD-39).
     * @minimum 0
     */
  basis_seq: number;
  /** UUIDv7 клиента; повтор с тем же id возвращает прежний ответ (AD-7). У подписанной команды — event_id пакета. */
  command_id: string;
  /** Протокол допуска с закрытым маршрутом. */
  document_id: string;
  /** @maxLength 128 */
  passport_id: string;
  /**
     * Версия политики, по которой показаны права (policy_seq сеанса); изменилась — 409 journal.stale_policy (AD-39).
     * @minimum 0
     */
  policy_seq: number;
  previous_passport_id?: string;
  recipe_ref: string;
  /** Подписанный пакет DSSE для операций с уровнем подписи ≥ 1 (AD-10, AD-13, AD-14): подписывает агент токена, сервер сверяет отпечаток. */
  signature?: DsseEnvelope;
  stage: AdmitPassportStage;
  /**
     * @minimum 0
     * @maximum 4
     */
  trust_level: number;
  versions: AdmitPassportVersions;
  /**
     * Рабочее место сеанса (барьер 2, AD-15).
     * @maxLength 128
     */
  workplace_id?: string;
}

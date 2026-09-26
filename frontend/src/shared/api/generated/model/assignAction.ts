/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AssignActionActionType } from './assignActionActionType';
import type { AssignActionDirection } from './assignActionDirection';
import type { DsseEnvelope } from './dsseEnvelope';
import type { EffectivenessPlanInput } from './effectivenessPlanInput';

export interface AssignAction {
  action_type: AssignActionActionType;
  /**
     * seq, на котором клиент видел объект (basis_seq из ответа чтения): после него в потоках гарда не должно быть новых записей guard_relevant, иначе 409 journal.stale_state (AD-39).
     * @minimum 0
     */
  basis_seq: number;
  /** UUIDv7 клиента; повтор с тем же id возвращает прежний ответ (AD-7). У подписанной команды — event_id пакета. */
  command_id: string;
  direction: AssignActionDirection;
  due_at?: string;
  /** План проверки эффективности (FR-64): метрика, базовый уровень, окно, критерий успеха; без него мера не создаётся (422 incident.effectiveness_plan_required). */
  effectiveness_plan: EffectivenessPlanInput;
  owner_id: string;
  /**
     * Версия политики, по которой показаны права (policy_seq сеанса); изменилась — 409 journal.stale_policy (AD-39).
     * @minimum 0
     */
  policy_seq: number;
  /** Подписанный пакет DSSE для операций с уровнем подписи ≥ 1 (AD-10, AD-13, AD-14): подписывает агент токена, сервер сверяет отпечаток. */
  signature?: DsseEnvelope;
  /**
     * Предложение, из которого родилась мера.
     * @maxLength 128
     */
  suggestion_id?: string;
  /**
     * Что делается словами — основа организационной памяти.
     * @maxLength 256
     */
  title?: string;
  /**
     * Рабочее место сеанса (барьер 2, AD-15).
     * @maxLength 128
     */
  workplace_id?: string;
}

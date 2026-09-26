/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DsseEnvelope } from './dsseEnvelope';
import type { StartRunMode } from './startRunMode';
import type { StartRunStart } from './startRunStart';

export interface StartRun {
  /**
     * seq, на котором клиент видел объект (basis_seq из ответа чтения): после него в потоках гарда не должно быть новых записей guard_relevant, иначе 409 journal.stale_state (AD-39).
     * @minimum 0
     */
  basis_seq: number;
  /** UUIDv7 клиента; повтор с тем же id возвращает прежний ответ (AD-7). У подписанной команды — event_id пакета. */
  command_id: string;
  /**
     * Шаг старта на заготовках (профиль fixtures): курсор сразу на этом шаге, шаги до него пройдены; сильнее start. Живой прогон шагов не нумерует — отказ api.validation_failed.
     * @minimum 0
     */
  from_step?: number;
  /**
     * Изделий сценария; пусто — по определению.
     * @minimum 0
     * @maximum 10000
     */
  items?: number;
  /** interactive — решения на столах ролей; autocheck — demo-signer (AD-26). */
  mode: StartRunMode;
  /**
     * Версия политики, по которой показаны права (policy_seq сеанса); изменилась — 409 journal.stale_policy (AD-39).
     * @minimum 0
     */
  policy_seq: number;
  /**
     * Seed генератора; пусто — из определения сценария.
     * @minimum 0
     */
  seed?: number;
  /** Подписанный пакет DSSE для операций с уровнем подписи ≥ 1 (AD-10, AD-13, AD-14): подписывает агент токена, сервер сверяет отпечаток. */
  signature?: DsseEnvelope;
  /**
     * Ускорение ×1…×1000; пусто — ×1.
     * @minimum 1
     * @maximum 1000
     */
  speed?: number;
  /** Откуда начинать: beginning — с самого начала (по умолчанию); start_step — с точки старта сценария (start_step в simulation.scenario.list): история сразу «у катастрофы», шаги до точки считаются пройденными. Пока только заготовки (профиль fixtures). */
  start?: StartRunStart;
  /**
     * Рабочее место сеанса (барьер 2, AD-15).
     * @maxLength 128
     */
  workplace_id?: string;
}

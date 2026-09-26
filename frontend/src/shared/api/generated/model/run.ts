/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { RunMode } from './runMode';
import type { RunState } from './runState';
import type { RunWait } from './runWait';

export interface Run {
  /** seq, на котором построен ответ (AD-39). */
  basis_seq: number;
  /**
     * Утверждений табло совпало.
     * @minimum 0
     */
  board_passed: number;
  /** @minimum 0 */
  board_total: number;
  /** Доменное «сейчас» прогона — последняя запись time.clock.ticked (AD-37). */
  clock_at: string;
  /**
     * null — прогон идёт.
     * @nullable
     */
  finished_at: string | null;
  /**
     * Изделий сценария (без фоновых).
     * @minimum 0
     */
  items: number;
  /** interactive — ждёт решения на столах ролей; autocheck — решения подписывает demo-signer (AD-26). */
  mode: RunMode;
  run_id: string;
  scenario_id: string;
  scenario_version: string;
  /** @minimum 0 */
  seed: number;
  /**
     * Ускорение доменных часов ×1…×1000.
     * @minimum 1
     * @maximum 1000
     */
  speed: number;
  started_at: string;
  state: RunState;
  /**
     * Номер шага сценария (на заготовках — шаг курсора, AD-36).
     * @minimum 0
     */
  step: number;
  /**
     * Всего шагов.
     * @minimum 0
     */
  steps: number;
  /** На каком решении человека стоит сценарий. */
  waiting_for?: RunWait;
}

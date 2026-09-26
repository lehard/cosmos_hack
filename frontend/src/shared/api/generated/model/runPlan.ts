/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { PlanEntry } from './planEntry';
import type { RunPlanState } from './runPlanState';
import type { RunWait } from './runWait';

export interface RunPlan {
  /** Доменное «сейчас» прогона (AD-37): часы в шапке столов. */
  clock_at: string;
  items: PlanEntry[];
  /** Начало живой части прогона (сценарий показа): раньше — готовая история, её прогон проигрывает сразу. */
  live_from?: string;
  run_id: string;
  /**
     * Ускорение доменных часов ×1…×1000.
     * @minimum 1
     * @maximum 1000
     */
  speed: number;
  state: RunPlanState;
  /** Чьё решение ждёт прогон сейчас. */
  waiting_for?: RunWait;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ActionHistory } from './actionHistory';
import type { CorrectiveActionViewActionType } from './correctiveActionViewActionType';
import type { CorrectiveActionViewDirection } from './correctiveActionViewDirection';
import type { CorrectiveActionViewStatus } from './correctiveActionViewStatus';
import type { EffectivenessPlanView } from './effectivenessPlanView';

export interface CorrectiveActionView {
  action_id: string;
  /** Коррекция / корректирующее / предупреждающее действие (ГОСТ Р ИСО 9000). */
  action_type: CorrectiveActionViewActionType;
  assigned_at: string;
  basis_seq: number;
  cause_category?: string;
  /** Попытка: 1 + сколько раз мера не помогла. */
  cycle: number;
  /** «Почему возник» / «почему пропустили». */
  direction: CorrectiveActionViewDirection;
  due_at?: string;
  /** С какого момента можно поставить «эффективно»: внедрено + окно наблюдения. */
  evaluation_due_at?: string;
  factor?: string;
  /** overdue — просрочена; ineffective — не помогла; hanging_temporary_control — висит временно усиленный контроль; evaluation_due — пора оценить; awaiting_window — идёт окно наблюдения. */
  flags: string[];
  history: ActionHistory[];
  implemented_at?: string;
  incident_id: string;
  owner: string;
  plan: EffectivenessPlanView;
  /** Назначена / внедрена (идёт окно наблюдения) / эффективна / переоткрыта — не помогла. «Внедрено» ≠ «эффективно». */
  status: CorrectiveActionViewStatus;
  suggestion_id?: string;
  title?: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCDecisionAction } from './nCDecisionAction';

export interface NCToDecide {
  /** Решения карточки: доступность для вошедшего (те же гарды, что у команд), почему, и последствия, вычисленные сервером из состояния изделия и политики. Интерфейс показывает только их. */
  actions?: NCDecisionAction[];
  /** Ремонт и «как есть» — только с действующим разрешением на отклонение. */
  concession_required: boolean;
  decision_due_at?: string;
  /** id операций решений, допустимых по состоянию (права — access.permission.list). */
  decisions: string[];
}

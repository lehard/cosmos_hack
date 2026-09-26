/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ActionHistoryResult } from './actionHistoryResult';
import type { ActionHistoryType } from './actionHistoryType';

export interface ActionHistory {
  actor?: string;
  at: string;
  cycle: number;
  event_id: string;
  evidence?: string;
  result?: ActionHistoryResult;
  text?: string;
  type: ActionHistoryType;
}

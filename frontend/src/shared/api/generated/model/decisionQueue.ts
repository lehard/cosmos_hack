/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DecisionQueueRow } from './decisionQueueRow';

export interface DecisionQueue {
  items: DecisionQueueRow[];
  next_cursor?: string;
}

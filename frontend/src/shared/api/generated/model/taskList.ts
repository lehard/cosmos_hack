/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { TaskEntry } from './taskEntry';

export interface TaskList {
  items: TaskEntry[];
  next_cursor?: string;
}

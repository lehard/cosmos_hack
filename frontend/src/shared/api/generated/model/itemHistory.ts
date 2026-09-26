/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ItemHistoryEntry } from './itemHistoryEntry';

export interface ItemHistory {
  items: ItemHistoryEntry[];
  next_cursor?: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { StoppedItem } from './stoppedItem';

export interface StoppedItemList {
  items: StoppedItem[];
  next_cursor?: string;
}

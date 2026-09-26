/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ItemRow } from './itemRow';

export interface ItemList {
  items: ItemRow[];
  next_cursor?: string;
}

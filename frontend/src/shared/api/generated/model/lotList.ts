/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { Lot } from './lot';

export interface LotList {
  items: Lot[];
  next_cursor?: string;
}

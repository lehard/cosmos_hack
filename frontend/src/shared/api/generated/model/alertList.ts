/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AlertEntry } from './alertEntry';

export interface AlertList {
  items: AlertEntry[];
  next_cursor?: string;
}

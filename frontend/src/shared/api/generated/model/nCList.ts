/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCSummary } from './nCSummary';

export interface NCList {
  items: NCSummary[];
  next_cursor?: string;
}

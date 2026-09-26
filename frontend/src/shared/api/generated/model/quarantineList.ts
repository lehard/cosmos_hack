/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { QuarantineEntry } from './quarantineEntry';

export interface QuarantineList {
  items: QuarantineEntry[];
  next_cursor?: string;
}

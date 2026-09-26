/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AccessGrantEntry } from './accessGrantEntry';

export interface AccessGrantHistory {
  items: AccessGrantEntry[];
  next_cursor?: string;
}

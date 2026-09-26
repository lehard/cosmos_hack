/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { KeyView } from './keyView';

export interface KeyList {
  items: KeyView[];
  next_cursor?: string;
}

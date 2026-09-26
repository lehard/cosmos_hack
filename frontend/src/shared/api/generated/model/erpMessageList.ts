/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ErpMessage } from './erpMessage';

export interface ErpMessageList {
  items: ErpMessage[];
  next_cursor?: string;
}

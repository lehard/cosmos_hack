/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ErpOrder } from './erpOrder';

export interface ErpOrderList {
  items: ErpOrder[];
  next_cursor?: string;
}

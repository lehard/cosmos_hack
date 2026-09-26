/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { TraceItem } from './traceItem';

export interface Trace {
  group_id?: string;
  heat_no?: string;
  items: TraceItem[];
  lot_id?: string;
}

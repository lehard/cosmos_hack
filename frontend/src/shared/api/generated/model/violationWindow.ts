/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DrillRef } from './drillRef';

export interface ViolationWindow {
  deviation_event_ids: string[];
  equipment_id: string;
  items: DrillRef[];
  nonconformities: DrillRef[];
  operation_run_ids: string[];
  step_key: string;
  /**
     * null — окно не закрыто.
     * @nullable
     */
  window_end: string | null;
  window_start: string;
}

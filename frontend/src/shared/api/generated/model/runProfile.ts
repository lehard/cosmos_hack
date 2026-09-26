/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { CycleParameter } from './cycleParameter';
import type { EquipmentEventRow } from './equipmentEventRow';
import type { RunProfileIntervalOrigin } from './runProfileIntervalOrigin';

export interface RunProfile {
  equipment_id?: string;
  events: EquipmentEventRow[];
  /** @nullable */
  finished_at: string | null;
  /** Происхождение интервала («Длительности»). */
  interval_origin: RunProfileIntervalOrigin;
  item_id: string;
  operation_run_id: string;
  /**
     * Псевдоним исполнителя; null — неизвестно.
     * @nullable
     */
  operator_id: string | null;
  parameters: CycleParameter[];
  program_ref?: string;
  special_process: boolean;
  started_at: string;
  step_key: string;
  tool_id?: string;
  /** Нарушение режима на специальном процессе (FR-151). */
  violation: boolean;
}

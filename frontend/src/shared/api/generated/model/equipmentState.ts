/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { EquipmentStateCondition } from './equipmentStateCondition';
import type { EquipmentStateControllerMode } from './equipmentStateControllerMode';
import type { EquipmentStateExecution } from './equipmentStateExecution';
import type { EquipmentStateSourceKind } from './equipmentStateSourceKind';
import type { EquipmentVerification } from './equipmentVerification';
import type { EquipmentWarning } from './equipmentWarning';

export interface EquipmentState {
  condition: EquipmentStateCondition;
  controller_mode: EquipmentStateControllerMode;
  /** Текущее выполнение операции (привязку делает только межизделийная стадия, AD-29). */
  current_run_id?: string;
  equipment_id: string;
  execution: EquipmentStateExecution;
  program_ref?: string;
  /** Источник данных (FR-140). */
  source_kind?: EquipmentStateSourceKind;
  /** Выполняет специальный процесс (FR-151). */
  special_process: boolean;
  station_id?: string;
  title: string;
  tool_id?: string;
  /**
     * @minimum 0
     * @nullable
     */
  tool_life_limit?: number | null;
  /**
     * @minimum 0
     * @nullable
     */
  tool_life_used?: number | null;
  updated_at?: string;
  verification: EquipmentVerification;
  warnings: EquipmentWarning[];
}

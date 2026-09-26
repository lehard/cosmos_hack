/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { EquipmentEventRow } from './equipmentEventRow';

export interface EquipmentTimeline {
  equipment_id: string;
  from: string;
  rows: EquipmentEventRow[];
  to: string;
}

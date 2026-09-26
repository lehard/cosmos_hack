/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { EquipmentEventRowLayer } from './equipmentEventRowLayer';
import type { EquipmentEventRowParams } from './equipmentEventRowParams';
import type { EquipmentEventRowSourceKind } from './equipmentEventRowSourceKind';

export interface EquipmentEventRow {
  ended_at?: string;
  event_id: string;
  event_type: string;
  /** Что делал станок / чем / как шёл процесс / отклонения. */
  layer: EquipmentEventRowLayer;
  occurred_at: string;
  params?: EquipmentEventRowParams;
  seq: number;
  source_kind?: EquipmentEventRowSourceKind;
  summary: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { EquipmentWarningKind } from './equipmentWarningKind';

export interface EquipmentWarning {
  event_id?: string;
  kind: EquipmentWarningKind;
  since: string;
  /** Подпись: «ресурс инструмента 73/75». */
  text: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { InspectionDefectSeverity } from './inspectionDefectSeverity';

export interface InspectionDefect {
  component_id?: string;
  defect_type_code?: string;
  description?: string;
  location?: string;
  severity: InspectionDefectSeverity;
  zone_id?: string;
}

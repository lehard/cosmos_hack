/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { InspectionMeasurementSourceVerdict } from './inspectionMeasurementSourceVerdict';
import type { InspectionMeasurementVerdict } from './inspectionMeasurementVerdict';

export interface InspectionMeasurement {
  characteristic: string;
  source_verdict: InspectionMeasurementSourceVerdict;
  tolerance?: string;
  value?: string;
  /** Строже из оценки источника и расчёта системы. */
  verdict: InspectionMeasurementVerdict;
}

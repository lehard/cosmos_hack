/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { InspectionDefect } from './inspectionDefect';
import type { InspectionResultMethod } from './inspectionResultMethod';
import type { InspectionResultOutcome } from './inspectionResultOutcome';
import type { InspectionResultProcessingState } from './inspectionResultProcessingState';
import type { InspectionResultReliability } from './inspectionResultReliability';
import type { InspectionResultSourceKind } from './inspectionResultSourceKind';

export interface InspectionResult {
  /**
     * @minimum 0
     * @maximum 10000
     */
  analyzer_confidence_bp?: number;
  defects: InspectionDefect[];
  event_id: string;
  evidence_refs: string[];
  item_id: string;
  limitations?: string[];
  method: InspectionResultMethod;
  /**
     * @minimum 0
     * @maximum 10000
     */
  observation_quality_bp?: number;
  occurred_at: string;
  operation_run_id?: string;
  /** «Оценка невозможна» ≠ «годно». */
  outcome: InspectionResultOutcome;
  processing_state: InspectionResultProcessingState;
  reliability: InspectionResultReliability;
  seq?: number;
  /** Пометка источника (FR-140). */
  source_kind: InspectionResultSourceKind;
  step_key: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { InspectionDefect } from './inspectionDefect';
import type { InspectionMeasurement } from './inspectionMeasurement';
import type { InspectionResultMethod } from './inspectionResultMethod';
import type { InspectionResultOutcome } from './inspectionResultOutcome';
import type { InspectionResultProcessingState } from './inspectionResultProcessingState';
import type { InspectionResultReliability } from './inspectionResultReliability';
import type { InspectionResultSourceKind } from './inspectionResultSourceKind';
import type { QualityAnalyzerStage } from './qualityAnalyzerStage';

export interface InspectionResult {
  /** Наблюдение анализатора (VisionQC). */
  analyzer?: boolean;
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
  /** Измерения «значение против допуска» (FR-36). */
  measurements?: InspectionMeasurement[];
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
  /** Почему система изменила исход: processing_not_completed, poor_observation, measurement_outside, not_measured. */
  reinterpreted?: string;
  reliability: InspectionResultReliability;
  /** Исход, сообщённый источником. */
  reported_outcome?: string;
  seq?: number;
  /** Пометка источника (FR-140). */
  source_kind: InspectionResultSourceKind;
  /** Ступени анализатора (FR-38). */
  stages?: QualityAnalyzerStage[];
  step_key: string;
  /** Наблюдение исправлено более поздней записью (FR-122). */
  superseded?: boolean;
  /**
     * Уровень доверия паспорта анализатора на момент наблюдения (AD-29).
     * @minimum 0
     * @maximum 4
     */
  trust_level?: number;
  /** admitted, no_qualified_analyzer, suspended:…, shadow, analyzer_version_mismatch. */
  trust_note?: string;
  /** Код причины «оценка невозможна». */
  unable_reason?: string;
}

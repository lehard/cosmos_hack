/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { CoveragePoint } from './coveragePoint';
import type { DefectTypeCoverage } from './defectTypeCoverage';
import type { InspectionCoverageQualityState } from './inspectionCoverageQualityState';

export interface InspectionCoverage {
  basis_seq: number;
  /** Все обязательные результаты получены. */
  complete: boolean;
  item_id: string;
  points: CoveragePoint[];
  /** Состояние качества изделия. */
  quality_state?: InspectionCoverageQualityState;
  types?: DefectTypeCoverage[];
}

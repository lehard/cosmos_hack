/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DefectTypeCoverageSeverity } from './defectTypeCoverageSeverity';
import type { DefectTypeCoverageStatus } from './defectTypeCoverageStatus';

export interface DefectTypeCoverage {
  code: string;
  event_ids?: string[];
  methods?: string[];
  name: string;
  severity: DefectTypeCoverageSeverity;
  /** not_checked — не проверено методом, способным выявить: не годность. */
  status: DefectTypeCoverageStatus;
}

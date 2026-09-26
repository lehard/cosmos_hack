/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { CoveragePointMethod } from './coveragePointMethod';
import type { CoveragePointMissingReason } from './coveragePointMissingReason';
import type { CoveragePointOutcome } from './coveragePointOutcome';
import type { CoveragePointStatus } from './coveragePointStatus';

export interface CoveragePoint {
  /** Результат или запись о пропуске. */
  event_id?: string;
  inspection_point: string;
  method: CoveragePointMethod;
  missing_reason?: CoveragePointMissingReason;
  /** Действующий исход полученного результата; «оценка невозможна» — нужен повторный контроль (FR-36). */
  outcome?: CoveragePointOutcome;
  required: boolean;
  /** Результат получен / ещё ждём / нет (quality.inspection.missing). */
  status: CoveragePointStatus;
  step_key: string;
}

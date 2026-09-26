/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { QualityAnalyzerStage } from './qualityAnalyzerStage';
import type { QualitySignalBasisKind } from './qualitySignalBasisKind';
import type { QualitySignalReactionOutcome } from './qualitySignalReactionOutcome';
import type { QualitySignalSeverity } from './qualitySignalSeverity';
import type { QualitySignalState } from './qualitySignalState';
import type { QualitySignalVersions } from './qualitySignalVersions';

export interface QualitySignal {
  /**
     * Уверенность анализатора, б. п. — не вероятность брака.
     * @minimum 0
     * @maximum 10000
     */
  analyzer_confidence_bp?: number;
  basis_kind: QualitySignalBasisKind;
  /** seq, на котором построен ответ (AD-39). */
  basis_seq: number;
  defect_id?: string;
  defect_type_code?: string;
  defect_type_known: boolean;
  /** Адреса материалов: кадр, иллюстрация (если есть). */
  evidence_refs: string[];
  item_id: string;
  item_label: string;
  /** Исходный результат контроля. */
  observation_event_id?: string;
  /**
     * Качество наблюдения, б. п.
     * @minimum 0
     * @maximum 10000
     */
  observation_quality_bp?: number;
  raised_at: string;
  reaction_map_ref: string;
  /** Исход по карте реакций. */
  reaction_outcome: QualitySignalReactionOutcome;
  severity: QualitySignalSeverity;
  signal_id: string;
  stages: QualityAnalyzerStage[];
  /** Рассмотрен ли сигнал человеком. */
  state: QualitySignalState;
  step_key: string;
  /**
     * Уровень доверия паспорта анализатора (AD-29).
     * @minimum 0
     * @maximum 4
     */
  trust_level?: number;
  /** Вектор версий наблюдения (AD-29): ревизия изделия, карта контроля, камера, калибровка, анализатор, профиль порогов, контракт, приложение. */
  versions?: QualitySignalVersions;
  zone_id?: string;
}

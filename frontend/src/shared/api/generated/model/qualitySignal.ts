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
  /**
     * Режим автоматизации, с которым исполнена реакция (FR-50).
     * @minimum 0
     * @maximum 5
     */
  automation_mode?: number;
  basis_kind: QualitySignalBasisKind;
  /** seq, на котором построен ответ (AD-39). */
  basis_seq: number;
  /** «Оценка невозможна» или пропуск проверки закрыты этим результатом повторного контроля (человек сигнал не рассматривал). */
  closed_by_event_id?: string;
  /** Предложенный уровень сдерживания (ось nonconformity). */
  containment?: string;
  defect_id?: string;
  defect_type_code?: string;
  defect_type_known: boolean;
  /** Адреса материалов: кадр, иллюстрация (если есть). */
  evidence_refs: string[];
  item_id: string;
  item_label: string;
  /** Почему реакция ограничена: mode_requires_human, trust_level_N, permissive_not_delegated, critical_hold. */
  limits?: string[];
  /** Исходный результат контроля. */
  observation_event_id?: string;
  /** Все наблюдения сигнала: повторные наблюдения одного дефекта (FR-37). */
  observation_ids?: string[];
  /**
     * Качество наблюдения, б. п.
     * @minimum 0
     * @maximum 10000
     */
  observation_quality_bp?: number;
  /** Реакция карты до ограничений, если она ограничена (FR-144). */
  proposed_outcome?: string;
  raised_at: string;
  reaction_map_ref: string;
  /** Исход по карте реакций. */
  reaction_outcome: QualitySignalReactionOutcome;
  /** Требование КД; нет — вопрос технологу, а не брак (FR-48). */
  requirement_ref?: string;
  /** Правило карты реакций. */
  rule_title?: string;
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
  /** Код причины «оценка невозможна». */
  unable_reason?: string;
  /** Сигнал «оценка невозможна» — повторный контроль, а не признак дефекта (FR-36). */
  unable_to_assess?: boolean;
  /** Вектор версий наблюдения (AD-29): ревизия изделия, карта контроля, камера, калибровка, анализатор, профиль порогов, контракт, приложение. */
  versions?: QualitySignalVersions;
  zone_id?: string;
}

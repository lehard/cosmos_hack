/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AnalyzerSummaryKind } from './analyzerSummaryKind';
import type { AnalyzerSummaryStage } from './analyzerSummaryStage';
import type { AnalyzerSummaryStatus } from './analyzerSummaryStatus';
import type { AnalyzerSummaryVersions } from './analyzerSummaryVersions';

export interface AnalyzerSummary {
  analyzer_id: string;
  /** Визуальный контроль или контроль действий оператора. */
  kind: AnalyzerSummaryKind;
  /** Действующий паспорт допуска. */
  passport_id?: string;
  /** Стадия допуска: тень, пилот, работа. */
  stage?: AnalyzerSummaryStage;
  status: AnalyzerSummaryStatus;
  title: string;
  /**
     * Уровень доверия паспорта → допустимые автоматические действия (AD-29).
     * @minimum 0
     * @maximum 4
     */
  trust_level?: number;
  /** Вектор версий (AD-29). */
  versions?: AnalyzerSummaryVersions;
}

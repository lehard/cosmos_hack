/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AnalyzerMonitor } from './analyzerMonitor';
import type { AnalyzerPassportAnalyzerKind } from './analyzerPassportAnalyzerKind';
import type { AnalyzerPassportStage } from './analyzerPassportStage';
import type { AnalyzerPassportStatus } from './analyzerPassportStatus';
import type { AnalyzerPassportVersions } from './analyzerPassportVersions';
import type { AnalyzerStatus } from './analyzerStatus';
import type { AnalyzerSuspension } from './analyzerSuspension';

export interface AnalyzerPassport {
  admitted_at: string;
  /** Допустимые автоматические действия уровня доверия (contracts/analyzer-trust-levels.yaml). */
  allowed_auto_actions: string[];
  analyzer_id: string;
  /** Визуальный контроль или контроль действий оператора. */
  analyzer_kind?: AnalyzerPassportAnalyzerKind;
  /** seq, на котором построен ответ (AD-39). */
  basis_seq: number;
  /** Протокол допуска (закрытый маршрут подписей). */
  document_id: string;
  /** Смены статуса: допуск, приостановка, возврат, вывод. */
  history?: AnalyzerStatus[];
  /** Контроль дрейфа правила автоотката (FR-101). */
  monitor?: AnalyzerMonitor;
  passport_id: string;
  previous_passport_id?: string;
  /** Происхождение записи допуска (AD-2): genesis — демо-затравка без экзамена (не промышленная валидация), personal — решение людей. */
  provenance?: string;
  /** Карта контроля. */
  recipe_ref: string;
  stage: AnalyzerPassportStage;
  status: AnalyzerPassportStatus;
  suspension?: AnalyzerSuspension;
  /** Название анализатора. */
  title?: string;
  /**
     * @minimum 0
     * @maximum 4
     */
  trust_level: number;
  /** Вектор версий допуска. */
  versions: AnalyzerPassportVersions;
}

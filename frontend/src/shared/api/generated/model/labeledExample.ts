/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { LabeledExampleVerdict } from './labeledExampleVerdict';
import type { LabeledExampleVersions } from './labeledExampleVersions';

export interface LabeledExample {
  /**
     * @minimum 0
     * @maximum 10000
     */
  analyzer_confidence_bp?: number;
  analyzer_defect?: string;
  analyzer_outcome: string;
  decided_at: string;
  decision_event_id: string;
  expert_defect?: string;
  expert_reason?: string;
  /** Отмечен как кандидат в размеченные данные для настройки модели. */
  for_adaptation: boolean;
  item_id?: string;
  observation_event_id: string;
  /** Эксперт подтвердил признак или отклонил (ложная тревога). */
  verdict: LabeledExampleVerdict;
  versions: LabeledExampleVersions;
}

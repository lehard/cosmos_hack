/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ObservationAccountStatusNow } from './observationAccountStatusNow';
import type { ObservationAccountStatusThen } from './observationAccountStatusThen';
import type { ObservationAccountVersions } from './observationAccountVersions';
import type { ObservationStage } from './observationStage';

export interface ObservationAccount {
  allowed_auto_actions: string[];
  /**
     * @minimum 0
     * @maximum 10000
     */
  confidence_bp?: number;
  event_id: string;
  item_id?: string;
  /**
     * Уровень доверия, с которым система реагировала.
     * @minimum 0
     * @maximum 4
     */
  level_then: number;
  missing_versions?: string[];
  occurred_at: string;
  outcome: string;
  passport_id?: string;
  point?: string;
  /**
     * @minimum 0
     * @maximum 10000
     */
  quality_bp?: number;
  /** Почему — по-русски, по шагам. */
  reasons: string[];
  stages?: ObservationStage[];
  status_now: ObservationAccountStatusNow;
  /** Статус паспорта на момент наблюдения. */
  status_then: ObservationAccountStatusThen;
  /** Изделие поставлено «под подозрением» по этому наблюдению. */
  suspicious: boolean;
  /** Вектор версий наблюдения (AD-29). */
  versions: ObservationAccountVersions;
}

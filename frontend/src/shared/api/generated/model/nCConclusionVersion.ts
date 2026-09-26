/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCConclusionVersionOutcome } from './nCConclusionVersionOutcome';
import type { NCRecordRef } from './nCRecordRef';

export interface NCConclusionVersion {
  /**
     * Режим автоматизации правила (FR-50).
     * @minimum 1
     * @maximum 5
     */
  automation_mode: number;
  causes: NCRecordRef[];
  event_id: string;
  outcome: NCConclusionVersionOutcome;
  recorded_at: string;
  /**
     * event_id записи, из-за которой вывод пересмотрен; null — первая версия.
     * @nullable
     */
  revised_due_to: string | null;
  rule_id: string;
  /** Ревизия нормативного слоя правила (FR-50): карта реакций, версия. */
  rule_rev?: string;
  /** @minimum 1 */
  version: number;
}

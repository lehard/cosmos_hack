/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { MissedObservation } from './missedObservation';
import type { RecheckItem } from './recheckItem';

export interface AdaptationEscape {
  analyzer_version?: string;
  defect_id: string;
  event_id: string;
  item_id: string;
  /** Метод способен выявить вид — иначе это не ошибка модели. */
  method_covers_defect: boolean;
  missed: MissedObservation[];
  recheck: RecheckItem[];
  recorded_at: string;
}

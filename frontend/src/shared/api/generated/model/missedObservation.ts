/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { MissedObservationVersions } from './missedObservationVersions';

export interface MissedObservation {
  event_id: string;
  item_id?: string;
  occurred_at: string;
  point?: string;
  /** Вектор версий наблюдения (AD-29). */
  versions: MissedObservationVersions;
}

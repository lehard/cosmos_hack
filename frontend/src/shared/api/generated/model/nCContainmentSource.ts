/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCContainmentSourceBy } from './nCContainmentSourceBy';
import type { NCContainmentSourceLevel } from './nCContainmentSourceLevel';

export interface NCContainmentSource {
  basis_gone: boolean;
  by: NCContainmentSourceBy;
  /** Запись-основание — её указывают в released_event_ids при снятии. */
  key: string;
  level: NCContainmentSourceLevel;
  reason: string;
  rule_id?: string;
}

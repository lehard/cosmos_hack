/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { FactorRef } from './factorRef';
import type { KnownGood } from './knownGood';
import type { ScopeItem } from './scopeItem';
import type { ScopeVersion } from './scopeVersion';
import type { TimeWindow } from './timeWindow';

export interface RiskScope {
  basis_seq: number;
  common_factor?: FactorRef;
  incident_id: string;
  incident_label: string;
  /** Изделия текущей версии. */
  items: ScopeItem[];
  last_known_good?: KnownGood;
  shipped_to_partners?: number;
  /** По возрастанию. */
  versions: ScopeVersion[];
  window?: TimeWindow;
}

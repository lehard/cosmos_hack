/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { Bottleneck } from './bottleneck';
import type { NodeAnomaly } from './nodeAnomaly';
import type { NodeCounters } from './nodeCounters';
import type { Period } from './period';

export interface NodeCounterSet {
  anomalies: NodeAnomaly[];
  basis_seq: number;
  bottleneck?: Bottleneck;
  counters: NodeCounters[];
  /** step_key узлов, где оценка невозможна (нет данных источника) — не «норма». */
  data_gaps: string[];
  period: Period;
  process_version_id: string;
}

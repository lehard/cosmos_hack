/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { Bottleneck } from './bottleneck';
import type { NodeAnomaly } from './nodeAnomaly';
import type { NodeCounters } from './nodeCounters';
import type { NodeCounterSetStepNames } from './nodeCounterSetStepNames';
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
  /** step_key → имя узла BPMN для всех узлов ответа (в том числе data_gaps); нет имени — ключа нет. */
  step_names?: NodeCounterSetStepNames;
}

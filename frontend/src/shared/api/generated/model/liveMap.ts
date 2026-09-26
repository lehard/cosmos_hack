/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { LiveMapStepNames } from './liveMapStepNames';
import type { MapBottleneck } from './mapBottleneck';
import type { MapIncident } from './mapIncident';
import type { MapItem } from './mapItem';
import type { MapNodeAnomaly } from './mapNodeAnomaly';
import type { MapNodeCounters } from './mapNodeCounters';
import type { MapVersionRef } from './mapVersionRef';

export interface LiveMap {
  anomalies: MapNodeAnomaly[];
  basis_seq: number;
  /** Нет — ограничение не выявлено. */
  bottleneck?: MapBottleneck;
  /** BPMN 2.0 XML показанной версии: BPMNDI, documentation, ant:properties/@stepKey. */
  bpmn_xml: string;
  counters: MapNodeCounters[];
  /** step_key узлов, где оценка невозможна: нет данных источника (не «норма»). */
  data_gaps: string[];
  incident?: MapIncident;
  items: MapItem[];
  /** Процесс показанной версии — id главного bpmn:process (UI-11). */
  process_id: string;
  /** Название процесса. */
  process_name: string;
  process_version: MapVersionRef;
  /** step_key → имя узла BPMN показанной версии для узлов ответа (в том числе data_gaps); нет имени — ключа нет. */
  step_names?: LiveMapStepNames;
  versions: MapVersionRef[];
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DrillRef } from './drillRef';
import type { MapNodeCounters } from './mapNodeCounters';
import type { NormRef } from './normRef';
import type { ProcessNodeCardKind } from './processNodeCardKind';
import type { ProcessNodeCardProperties } from './processNodeCardProperties';

export interface ProcessNodeCard {
  counters: MapNodeCounters;
  /** <bpmn:documentation> элемента. */
  documentation: string;
  element_id: string;
  /** Изделия в узле. */
  items: DrillRef[];
  kind: ProcessNodeCardKind;
  lane?: string;
  name: string;
  /** Открытые несоответствия узла. */
  nonconformities: DrillRef[];
  norm_refs: NormRef[];
  process_version_id: string;
  /** Наши свойства (ключи process.properties.*). */
  properties: ProcessNodeCardProperties;
  step_key: string;
}

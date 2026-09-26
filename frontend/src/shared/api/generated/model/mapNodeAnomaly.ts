/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { MapNodeAnomalyKind } from './mapNodeAnomalyKind';

export interface MapNodeAnomaly {
  kind: MapNodeAnomalyKind;
  step_key: string;
  /** Имя узла BPMN версии процесса; нет — показывать step_key. */
  step_name?: string;
  /** Порог текстом с единицей. */
  threshold?: string;
}

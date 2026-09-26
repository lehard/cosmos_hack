/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NodeAnomalyKind } from './nodeAnomalyKind';

export interface NodeAnomaly {
  kind: NodeAnomalyKind;
  step_key: string;
  /** Порог текстом с единицей. */
  threshold?: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { MetricValueOrigin } from './metricValueOrigin';

export interface MetricValue {
  /** Для длительностей: передано источником / вычислено системой. */
  origin?: MetricValueOrigin;
  /**
     * @minimum 0
     * @maximum 9
     */
  scale: number;
  /** Единица: pcs, bp (доли), s, min… */
  unit: string;
  /** Значение = value × 10^(−scale). */
  value: number;
}

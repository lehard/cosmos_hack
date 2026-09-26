/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type NodeAnomalyKind = typeof NodeAnomalyKind[keyof typeof NodeAnomalyKind];


export const NodeAnomalyKind = {
  queue_above_norm: 'queue_above_norm',
  wait_above_norm: 'wait_above_norm',
  downtime_over_threshold: 'downtime_over_threshold',
  output_spike: 'output_spike',
  defect_rate_out_of_control: 'defect_rate_out_of_control',
} as const;

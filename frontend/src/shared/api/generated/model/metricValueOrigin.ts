/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Для длительностей: передано источником / вычислено системой.
 */
export type MetricValueOrigin = typeof MetricValueOrigin[keyof typeof MetricValueOrigin];


export const MetricValueOrigin = {
  reported_by_source: 'reported_by_source',
  computed_by_system: 'computed_by_system',
  mixed: 'mixed',
} as const;

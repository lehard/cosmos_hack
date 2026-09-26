/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Раздел: раздельный учёт кейса §2.4, §5.2.
 */
export type MetricRowGroup = typeof MetricRowGroup[keyof typeof MetricRowGroup];


export const MetricRowGroup = {
  inspection: 'inspection',
  defects: 'defects',
  causes: 'causes',
  time: 'time',
  equipment: 'equipment',
  people: 'people',
  comparison: 'comparison',
} as const;

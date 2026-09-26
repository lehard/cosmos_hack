/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Происхождение интервала («Длительности»).
 */
export type RunProfileIntervalOrigin = typeof RunProfileIntervalOrigin[keyof typeof RunProfileIntervalOrigin];


export const RunProfileIntervalOrigin = {
  source_reported: 'source_reported',
  system_computed: 'system_computed',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Стоп точки процесса или критическая остановка.
 */
export type StationHoldLevel = typeof StationHoldLevel[keyof typeof StationHoldLevel];


export const StationHoldLevel = {
  process_point_stop: 'process_point_stop',
  critical_stop: 'critical_stop',
} as const;

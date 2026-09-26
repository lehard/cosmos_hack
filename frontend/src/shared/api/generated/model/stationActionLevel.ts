/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Уровень, с которым set остановит точку (поле команды).
 */
export type StationActionLevel = typeof StationActionLevel[keyof typeof StationActionLevel];


export const StationActionLevel = {
  process_point_stop: 'process_point_stop',
  critical_stop: 'critical_stop',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Статус в выбранном инциденте; нет — вне области (FR-9).
 */
export type MapItemIncidentStatus = typeof MapItemIncidentStatus[keyof typeof MapItemIncidentStatus];


export const MapItemIncidentStatus = {
  confirmed: 'confirmed',
  suspect: 'suspect',
  excluded: 'excluded',
  unknown: 'unknown',
} as const;

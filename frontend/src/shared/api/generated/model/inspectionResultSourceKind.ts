/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Пометка источника (FR-140).
 */
export type InspectionResultSourceKind = typeof InspectionResultSourceKind[keyof typeof InspectionResultSourceKind];


export const InspectionResultSourceKind = {
  manual_entry: 'manual_entry',
  machine: 'machine',
  sensor: 'sensor',
  camera: 'camera',
  external_system: 'external_system',
  import: 'import',
} as const;

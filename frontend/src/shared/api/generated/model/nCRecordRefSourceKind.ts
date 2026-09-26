/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Вид источника факта (FR-140).
 */
export type NCRecordRefSourceKind = typeof NCRecordRefSourceKind[keyof typeof NCRecordRefSourceKind];


export const NCRecordRefSourceKind = {
  manual_entry: 'manual_entry',
  machine: 'machine',
  sensor: 'sensor',
  camera: 'camera',
  external_system: 'external_system',
  import: 'import',
} as const;

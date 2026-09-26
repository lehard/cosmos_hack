/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type EquipmentEventRowSourceKind = typeof EquipmentEventRowSourceKind[keyof typeof EquipmentEventRowSourceKind];


export const EquipmentEventRowSourceKind = {
  manual_entry: 'manual_entry',
  machine: 'machine',
  sensor: 'sensor',
  camera: 'camera',
  external_system: 'external_system',
  import: 'import',
} as const;

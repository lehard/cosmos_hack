/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Источник данных (FR-140).
 */
export type EquipmentStateSourceKind = typeof EquipmentStateSourceKind[keyof typeof EquipmentStateSourceKind];


export const EquipmentStateSourceKind = {
  manual_entry: 'manual_entry',
  machine: 'machine',
  sensor: 'sensor',
  camera: 'camera',
  external_system: 'external_system',
  import: 'import',
} as const;

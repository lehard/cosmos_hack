/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Пометка источника факта (FR-140).
 */
export type PassportEntrySourceKind = typeof PassportEntrySourceKind[keyof typeof PassportEntrySourceKind];


export const PassportEntrySourceKind = {
  manual_entry: 'manual_entry',
  machine: 'machine',
  sensor: 'sensor',
  camera: 'camera',
  external_system: 'external_system',
  import: 'import',
} as const;

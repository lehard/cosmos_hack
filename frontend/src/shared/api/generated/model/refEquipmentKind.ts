/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type RefEquipmentKind = typeof RefEquipmentKind[keyof typeof RefEquipmentKind];


export const RefEquipmentKind = {
  cnc_machine: 'cnc_machine',
  welding_source: 'welding_source',
  camera: 'camera',
  cmm: 'cmm',
  leak_tester: 'leak_tester',
  torque_wrench: 'torque_wrench',
  xray: 'xray',
  test_bench: 'test_bench',
  other: 'other',
} as const;

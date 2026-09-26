/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type EquipmentStateCondition = typeof EquipmentStateCondition[keyof typeof EquipmentStateCondition];


export const EquipmentStateCondition = {
  normal: 'normal',
  warning: 'warning',
  fault: 'fault',
  unknown: 'unknown',
} as const;

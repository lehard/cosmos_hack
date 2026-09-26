/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type EquipmentWarningKind = typeof EquipmentWarningKind[keyof typeof EquipmentWarningKind];


export const EquipmentWarningKind = {
  out_of_setpoint: 'out_of_setpoint',
  overload: 'overload',
  tool_life_warning: 'tool_life_warning',
  manual_override: 'manual_override',
  unplanned_program_change: 'unplanned_program_change',
  alarm: 'alarm',
  verification_due: 'verification_due',
  other: 'other',
} as const;

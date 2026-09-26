/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type EquipmentVerificationStatus = typeof EquipmentVerificationStatus[keyof typeof EquipmentVerificationStatus];


export const EquipmentVerificationStatus = {
  valid: 'valid',
  expiring: 'expiring',
  expired: 'expired',
  unknown: 'unknown',
} as const;

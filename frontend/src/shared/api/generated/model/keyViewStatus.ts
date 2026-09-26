/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * unavailable — ключ недоступен: «не проверяемо», никогда не «валидно» (AD-32).
 */
export type KeyViewStatus = typeof KeyViewStatus[keyof typeof KeyViewStatus];


export const KeyViewStatus = {
  active: 'active',
  revoked: 'revoked',
  expired: 'expired',
  unavailable: 'unavailable',
} as const;

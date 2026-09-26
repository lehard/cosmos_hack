/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Учётная запись: нет, ждёт активации, действует, заблокирована (FR-128).
 */
export type AccessPersonAccountStatus = typeof AccessPersonAccountStatus[keyof typeof AccessPersonAccountStatus];


export const AccessPersonAccountStatus = {
  none: 'none',
  pending: 'pending',
  active: 'active',
  blocked: 'blocked',
} as const;

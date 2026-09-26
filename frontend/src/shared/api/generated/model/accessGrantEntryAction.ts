/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Выдано, отозвано, установлено.
 */
export type AccessGrantEntryAction = typeof AccessGrantEntryAction[keyof typeof AccessGrantEntryAction];


export const AccessGrantEntryAction = {
  granted: 'granted',
  revoked: 'revoked',
  set: 'set',
} as const;

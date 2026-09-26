/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * enabled — включена (реальная система), disabled — выключена, stand — стенд (эмулятор).
 */
export type IntegrationEntryState = typeof IntegrationEntryState[keyof typeof IntegrationEntryState];


export const IntegrationEntryState = {
  enabled: 'enabled',
  disabled: 'disabled',
  stand: 'stand',
} as const;

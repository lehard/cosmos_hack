/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * enabled — включить реальную систему; disabled — выключить; stand — переключить на стенд (в prod — отказ ops.stand_forbidden).
 */
export type SetIntegrationStateState = typeof SetIntegrationStateState[keyof typeof SetIntegrationStateState];


export const SetIntegrationStateState = {
  enabled: 'enabled',
  disabled: 'disabled',
  stand: 'stand',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * disabled — система не включена (integrations.enabled) или канал выключен.
 */
export type IntegrationStateState = typeof IntegrationStateState[keyof typeof IntegrationStateState];


export const IntegrationStateState = {
  ok: 'ok',
  degraded: 'degraded',
  disabled: 'disabled',
} as const;

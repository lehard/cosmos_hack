/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type IntegrationCheckResult = typeof IntegrationCheckResult[keyof typeof IntegrationCheckResult];


export const IntegrationCheckResult = {
  ok: 'ok',
  degraded: 'degraded',
  unreachable: 'unreachable',
  not_supported: 'not_supported',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Живое состояние канала обмена (сверка ответной стороны, AD-18).
 */
export type IntegrationEntryChannel = typeof IntegrationEntryChannel[keyof typeof IntegrationEntryChannel];


export const IntegrationEntryChannel = {
  ok: 'ok',
  degraded: 'degraded',
  disabled: 'disabled',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Состояние канала обмена.
 */
export type PartnerChannel = typeof PartnerChannel[keyof typeof PartnerChannel];


export const PartnerChannel = {
  ok: 'ok',
  degraded: 'degraded',
  unknown: 'unknown',
} as const;

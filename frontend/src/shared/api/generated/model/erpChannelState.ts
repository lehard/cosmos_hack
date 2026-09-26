/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ErpChannelState = typeof ErpChannelState[keyof typeof ErpChannelState];


export const ErpChannelState = {
  ok: 'ok',
  degraded: 'degraded',
  disabled: 'disabled',
} as const;

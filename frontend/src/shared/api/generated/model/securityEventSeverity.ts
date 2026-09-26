/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type SecurityEventSeverity = typeof SecurityEventSeverity[keyof typeof SecurityEventSeverity];


export const SecurityEventSeverity = {
  info: 'info',
  warning: 'warning',
  alarm: 'alarm',
} as const;

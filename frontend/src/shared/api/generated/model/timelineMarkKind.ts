/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type TimelineMarkKind = typeof TimelineMarkKind[keyof typeof TimelineMarkKind];


export const TimelineMarkKind = {
  escalation: 'escalation',
  process_stop: 'process_stop',
  spike: 'spike',
  revision: 'revision',
} as const;

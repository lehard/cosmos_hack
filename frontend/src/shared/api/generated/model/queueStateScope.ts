/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type QueueStateScope = typeof QueueStateScope[keyof typeof QueueStateScope];


export const QueueStateScope = {
  partition: 'partition',
  global: 'global',
  outbox: 'outbox',
} as const;

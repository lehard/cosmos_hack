/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type AcknowledgeTaskOutcome = typeof AcknowledgeTaskOutcome[keyof typeof AcknowledgeTaskOutcome];


export const AcknowledgeTaskOutcome = {
  done: 'done',
  accepted: 'accepted',
  declined: 'declined',
} as const;

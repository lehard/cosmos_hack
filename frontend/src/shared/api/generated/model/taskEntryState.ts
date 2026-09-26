/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type TaskEntryState = typeof TaskEntryState[keyof typeof TaskEntryState];


export const TaskEntryState = {
  open: 'open',
  done: 'done',
  accepted: 'accepted',
  declined: 'declined',
  withdrawn: 'withdrawn',
} as const;

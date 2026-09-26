/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type NotificationsTaskListState = typeof NotificationsTaskListState[keyof typeof NotificationsTaskListState];


export const NotificationsTaskListState = {
  open: 'open',
  done: 'done',
  accepted: 'accepted',
  declined: 'declined',
  withdrawn: 'withdrawn',
} as const;

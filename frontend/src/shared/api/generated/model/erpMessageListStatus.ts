/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ErpMessageListStatus = typeof ErpMessageListStatus[keyof typeof ErpMessageListStatus];


export const ErpMessageListStatus = {
  queued: 'queued',
  sent: 'sent',
  acknowledged: 'acknowledged',
  rejected: 'rejected',
  quarantined: 'quarantined',
} as const;

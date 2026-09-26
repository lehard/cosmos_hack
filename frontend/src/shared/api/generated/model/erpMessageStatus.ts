/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Словарь erp_message_status (не ось).
 */
export type ErpMessageStatus = typeof ErpMessageStatus[keyof typeof ErpMessageStatus];


export const ErpMessageStatus = {
  queued: 'queued',
  sent: 'sent',
  acknowledged: 'acknowledged',
  rejected: 'rejected',
  quarantined: 'quarantined',
} as const;

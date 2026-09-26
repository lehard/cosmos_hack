/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Состояние партии; блок партии — ось «сдерживание» nonconformity.
 */
export type LotStatus = typeof LotStatus[keyof typeof LotStatus];


export const LotStatus = {
  received: 'received',
  registered: 'registered',
  accepted: 'accepted',
  rejected: 'rejected',
  on_hold: 'on_hold',
  issued: 'issued',
  consumed: 'consumed',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * decision — решение человека; event — событие машины или внешней системы; alert — запланированный сбой (видно заранее); stand, tamper — служебное.
 */
export type PlanEntryKind = typeof PlanEntryKind[keyof typeof PlanEntryKind];


export const PlanEntryKind = {
  decision: 'decision',
  event: 'event',
  alert: 'alert',
  stand: 'stand',
  tamper: 'tamper',
} as const;

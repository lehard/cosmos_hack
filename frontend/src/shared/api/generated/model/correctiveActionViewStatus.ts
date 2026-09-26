/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Назначена / внедрена (идёт окно наблюдения) / эффективна / переоткрыта — не помогла. «Внедрено» ≠ «эффективно».
 */
export type CorrectiveActionViewStatus = typeof CorrectiveActionViewStatus[keyof typeof CorrectiveActionViewStatus];


export const CorrectiveActionViewStatus = {
  assigned: 'assigned',
  implemented: 'implemented',
  effective: 'effective',
  reopened: 'reopened',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Квитанция MES; пусто — ждём.
 */
export type MesBlockOutcome = typeof MesBlockOutcome[keyof typeof MesBlockOutcome];


export const MesBlockOutcome = {
  accepted: 'accepted',
  duplicate: 'duplicate',
  rejected: 'rejected',
} as const;

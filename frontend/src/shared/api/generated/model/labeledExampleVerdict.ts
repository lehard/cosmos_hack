/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Эксперт подтвердил признак или отклонил (ложная тревога).
 */
export type LabeledExampleVerdict = typeof LabeledExampleVerdict[keyof typeof LabeledExampleVerdict];


export const LabeledExampleVerdict = {
  confirmed: 'confirmed',
  rejected: 'rejected',
} as const;

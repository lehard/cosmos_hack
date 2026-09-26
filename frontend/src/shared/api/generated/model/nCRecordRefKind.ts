/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Вид записи (AD-2): исходный факт, вывод системы, решение человека.
 */
export type NCRecordRefKind = typeof NCRecordRefKind[keyof typeof NCRecordRefKind];


export const NCRecordRefKind = {
  fact: 'fact',
  reaction: 'reaction',
  decision: 'decision',
  service: 'service',
} as const;

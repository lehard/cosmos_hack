/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Вид записи (AD-2): исходный сигнал, анализ системы, решение человека.
 */
export type PassportEntryKind = typeof PassportEntryKind[keyof typeof PassportEntryKind];


export const PassportEntryKind = {
  fact: 'fact',
  reaction: 'reaction',
  decision: 'decision',
  service: 'service',
} as const;

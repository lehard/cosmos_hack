/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Вид записи (AD-2): исходный факт, вывод системы, решение человека, служебная.
 */
export type JournalEventViewKind = typeof JournalEventViewKind[keyof typeof JournalEventViewKind];


export const JournalEventViewKind = {
  fact: 'fact',
  reaction: 'reaction',
  decision: 'decision',
  service: 'service',
} as const;

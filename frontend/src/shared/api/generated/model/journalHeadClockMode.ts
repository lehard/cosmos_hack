/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Режим часов журнала (AD-37).
 */
export type JournalHeadClockMode = typeof JournalHeadClockMode[keyof typeof JournalHeadClockMode];


export const JournalHeadClockMode = {
  system: 'system',
  scenario: 'scenario',
} as const;

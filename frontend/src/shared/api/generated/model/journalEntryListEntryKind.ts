/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type JournalEntryListEntryKind = typeof JournalEntryListEntryKind[keyof typeof JournalEntryListEntryKind];


export const JournalEntryListEntryKind = {
  fact: 'fact',
  reaction: 'reaction',
  decision: 'decision',
  service: 'service',
} as const;

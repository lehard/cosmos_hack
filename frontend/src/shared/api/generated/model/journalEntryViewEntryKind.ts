/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type JournalEntryViewEntryKind = typeof JournalEntryViewEntryKind[keyof typeof JournalEntryViewEntryKind];


export const JournalEntryViewEntryKind = {
  fact: 'fact',
  reaction: 'reaction',
  decision: 'decision',
  service: 'service',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type AttentionEntryKind = typeof AttentionEntryKind[keyof typeof AttentionEntryKind];


export const AttentionEntryKind = {
  overdue_decision: 'overdue_decision',
  unverified_measures: 'unverified_measures',
  temporary_measures: 'temporary_measures',
} as const;

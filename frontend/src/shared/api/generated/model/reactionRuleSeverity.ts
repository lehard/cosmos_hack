/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ReactionRuleSeverity = typeof ReactionRuleSeverity[keyof typeof ReactionRuleSeverity];


export const ReactionRuleSeverity = {
  critical: 'critical',
  major: 'major',
  minor: 'minor',
  unknown: 'unknown',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ReactionRuleOutcome = typeof ReactionRuleOutcome[keyof typeof ReactionRuleOutcome];


export const ReactionRuleOutcome = {
  pass_to_next: 'pass_to_next',
  manual_review: 'manual_review',
  isolate: 'isolate',
  question_to_technologist: 'question_to_technologist',
} as const;

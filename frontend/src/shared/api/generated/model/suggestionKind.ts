/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type SuggestionKind = typeof SuggestionKind[keyof typeof SuggestionKind];


export const SuggestionKind = {
  bottleneck: 'bottleneck',
  risk_scope: 'risk_scope',
  reaction_rule_candidate: 'reaction_rule_candidate',
  analyzer_adaptation: 'analyzer_adaptation',
  data_deficit: 'data_deficit',
  other: 'other',
} as const;

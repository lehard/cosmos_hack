/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Класс действия (AD-27).
 */
export type ReactionRuleActionClass = typeof ReactionRuleActionClass[keyof typeof ReactionRuleActionClass];


export const ReactionRuleActionClass = {
  record: 'record',
  protective: 'protective',
  permissive: 'permissive',
  irreversible: 'irreversible',
} as const;

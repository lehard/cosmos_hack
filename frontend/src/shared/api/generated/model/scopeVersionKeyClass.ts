/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Класс ключа подписи сужения (AD-10).
 */
export type ScopeVersionKeyClass = typeof ScopeVersionKeyClass[keyof typeof ScopeVersionKeyClass];


export const ScopeVersionKeyClass = {
  personal: 'personal',
  device: 'device',
  server_attested: 'server_attested',
  scenario: 'scenario',
} as const;

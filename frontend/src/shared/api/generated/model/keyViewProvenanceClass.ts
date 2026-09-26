/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Класс доверия ключа (наследуется от регистрирующих подписей, AD-11).
 */
export type KeyViewProvenanceClass = typeof KeyViewProvenanceClass[keyof typeof KeyViewProvenanceClass];


export const KeyViewProvenanceClass = {
  personal: 'personal',
  device: 'device',
  server_attested: 'server_attested',
  scenario: 'scenario',
  genesis: 'genesis',
  partner: 'partner',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Класс происхождения подписи (AD-2).
 */
export type ItemSignatureClass = typeof ItemSignatureClass[keyof typeof ItemSignatureClass];


export const ItemSignatureClass = {
  device: 'device',
  personal: 'personal',
  paper: 'paper',
  partner: 'partner',
  server_attested: 'server_attested',
  scenario: 'scenario',
  genesis: 'genesis',
} as const;

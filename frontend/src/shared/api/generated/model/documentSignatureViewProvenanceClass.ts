/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Класс происхождения подписи (AD-2).
 */
export type DocumentSignatureViewProvenanceClass = typeof DocumentSignatureViewProvenanceClass[keyof typeof DocumentSignatureViewProvenanceClass];


export const DocumentSignatureViewProvenanceClass = {
  personal: 'personal',
  paper: 'paper',
  partner: 'partner',
  scenario: 'scenario',
  genesis: 'genesis',
  server_attested: 'server_attested',
  device: 'device',
} as const;

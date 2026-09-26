/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Класс подписи (AD-10): personal — ключ человека, server_attested — заверено сервером, paper — бумага с заверением, scenario — ключ сценария.
 */
export type CriticalActionSignerKeyClass = typeof CriticalActionSignerKeyClass[keyof typeof CriticalActionSignerKeyClass];


export const CriticalActionSignerKeyClass = {
  personal: 'personal',
  device: 'device',
  server_attested: 'server_attested',
  scenario: 'scenario',
  paper: 'paper',
  partner: 'partner',
  genesis: 'genesis',
} as const;

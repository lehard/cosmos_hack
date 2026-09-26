/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Внешняя система.
 */
export type IntegrationEntrySystem = typeof IntegrationEntrySystem[keyof typeof IntegrationEntrySystem];


export const IntegrationEntrySystem = {
  onec: 'onec',
  galaktika: 'galaktika',
  mes: 'mes',
  kompas: 'kompas',
  skud: 'skud',
  ca: 'ca',
  visionqc: 'visionqc',
  operatorvision: 'operatorvision',
  partner: 'partner',
} as const;

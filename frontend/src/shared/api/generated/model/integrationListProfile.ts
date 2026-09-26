/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * В prod стенд запрещён.
 */
export type IntegrationListProfile = typeof IntegrationListProfile[keyof typeof IntegrationListProfile];


export const IntegrationListProfile = {
  fixtures: 'fixtures',
  demo: 'demo',
  load: 'load',
  prod: 'prod',
} as const;

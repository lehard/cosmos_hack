/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * interactive — ждёт решения на столах ролей; autocheck — решения подписывает demo-signer (AD-26).
 */
export type RunMode = typeof RunMode[keyof typeof RunMode];


export const RunMode = {
  interactive: 'interactive',
  autocheck: 'autocheck',
  load: 'load',
} as const;

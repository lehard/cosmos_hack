/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * interactive — решения на столах ролей; autocheck — demo-signer (AD-26).
 */
export type StartRunMode = typeof StartRunMode[keyof typeof StartRunMode];


export const StartRunMode = {
  interactive: 'interactive',
  autocheck: 'autocheck',
  load: 'load',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Ось момента (AD-37): occurred — «как было» (по умолчанию), recorded — «что мы знали».
 */
export type Axis = typeof Axis[keyof typeof Axis];


export const Axis = {
  occurred: 'occurred',
  recorded: 'recorded',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */

/**
 * Ось момента (AD-37).
 */
export type Axis = typeof Axis[keyof typeof Axis];


export const Axis = {
  occurred: 'occurred',
  recorded: 'recorded',
} as const;

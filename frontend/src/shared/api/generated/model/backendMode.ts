/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */

/**
 * Режим ведущих портов (AD-36).
 */
export type BackendMode = typeof BackendMode[keyof typeof BackendMode];


export const BackendMode = {
  fixtures: 'fixtures',
  live: 'live',
} as const;

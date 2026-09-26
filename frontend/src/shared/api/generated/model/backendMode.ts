/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Режим ведущих портов, отдавших ответ (AD-36, FR-150).
 */
export type BackendMode = typeof BackendMode[keyof typeof BackendMode];


export const BackendMode = {
  fixtures: 'fixtures',
  live: 'live',
} as const;

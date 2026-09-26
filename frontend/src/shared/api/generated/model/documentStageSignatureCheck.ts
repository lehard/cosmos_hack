/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Проверка подписи (FR-68): демо без агента токена — unchecked.
 */
export type DocumentStageSignatureCheck = typeof DocumentStageSignatureCheck[keyof typeof DocumentStageSignatureCheck];


export const DocumentStageSignatureCheck = {
  valid: 'valid',
  rejected: 'rejected',
  not_verifiable: 'not_verifiable',
  unchecked: 'unchecked',
} as const;

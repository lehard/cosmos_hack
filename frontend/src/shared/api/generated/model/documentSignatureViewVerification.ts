/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Статус автоматической проверки подписи (FR-68); недоступный ключ — unverifiable, никогда не valid (AD-32).
 */
export type DocumentSignatureViewVerification = typeof DocumentSignatureViewVerification[keyof typeof DocumentSignatureViewVerification];


export const DocumentSignatureViewVerification = {
  valid: 'valid',
  invalid: 'invalid',
  unverifiable: 'unverifiable',
  pending: 'pending',
} as const;

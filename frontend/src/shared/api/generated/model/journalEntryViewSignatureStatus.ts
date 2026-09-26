/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Статус проверки подписи (FR-68); not_checked — профиль demo до эпика 05.
 */
export type JournalEntryViewSignatureStatus = typeof JournalEntryViewSignatureStatus[keyof typeof JournalEntryViewSignatureStatus];


export const JournalEntryViewSignatureStatus = {
  valid: 'valid',
  invalid: 'invalid',
  unverifiable: 'unverifiable',
  not_checked: 'not_checked',
} as const;

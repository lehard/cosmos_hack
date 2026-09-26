/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Класс хранения ключа (Д-72): hardware_token — физический ключ, software_browser — ключ в браузере под PIN.
 */
export type DocumentStageSignatureKeyStorage = typeof DocumentStageSignatureKeyStorage[keyof typeof DocumentStageSignatureKeyStorage];


export const DocumentStageSignatureKeyStorage = {
  hardware_token: 'hardware_token',
  software_browser: 'software_browser',
} as const;

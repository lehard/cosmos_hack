/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Класс хранения ключа подписанта (AD-14, Д-72): физический ключ или ключ в браузере.
 */
export type ItemSignatureKeyStorage = typeof ItemSignatureKeyStorage[keyof typeof ItemSignatureKeyStorage];


export const ItemSignatureKeyStorage = {
  hardware_token: 'hardware_token',
  software_browser: 'software_browser',
} as const;

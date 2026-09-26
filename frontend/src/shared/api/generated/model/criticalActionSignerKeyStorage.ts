/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Класс хранения ключа человека (AD-14, Д-72): физический ключ или ключ в браузере под PIN.
 */
export type CriticalActionSignerKeyStorage = typeof CriticalActionSignerKeyStorage[keyof typeof CriticalActionSignerKeyStorage];


export const CriticalActionSignerKeyStorage = {
  hardware_token: 'hardware_token',
  software_browser: 'software_browser',
} as const;

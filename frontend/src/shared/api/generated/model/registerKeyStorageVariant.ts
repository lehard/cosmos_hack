/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Где лежит ключ в браузере: расширение или хранилище страницы.
 */
export type RegisterKeyStorageVariant = typeof RegisterKeyStorageVariant[keyof typeof RegisterKeyStorageVariant];


export const RegisterKeyStorageVariant = {
  extension: 'extension',
  page: 'page',
} as const;

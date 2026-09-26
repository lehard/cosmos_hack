/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Статус проверки: цело / отвергнуто / не проверяемо / не проверялась (демо без подписей).
 */
export type ItemSignatureCheck = typeof ItemSignatureCheck[keyof typeof ItemSignatureCheck];


export const ItemSignatureCheck = {
  valid: 'valid',
  rejected: 'rejected',
  not_verifiable: 'not_verifiable',
  unchecked: 'unchecked',
} as const;

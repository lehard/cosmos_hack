/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Способ подписи решения: агент токена или ключ в браузере, бумага с заверением, демо-подписант (Д-59).
 */
export type ItemSignatureMethod = typeof ItemSignatureMethod[keyof typeof ItemSignatureMethod];


export const ItemSignatureMethod = {
  token_agent: 'token_agent',
  paper: 'paper',
  demo_signer: 'demo_signer',
} as const;

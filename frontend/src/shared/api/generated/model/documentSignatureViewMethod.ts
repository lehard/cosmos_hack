/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * source_decision — этап закрыт самим решением-источником (by_source).
 */
export type DocumentSignatureViewMethod = typeof DocumentSignatureViewMethod[keyof typeof DocumentSignatureViewMethod];


export const DocumentSignatureViewMethod = {
  token_agent: 'token_agent',
  paper: 'paper',
  device: 'device',
  demo_signer: 'demo_signer',
  source_decision: 'source_decision',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type SendExtractKind = typeof SendExtractKind[keyof typeof SendExtractKind];


export const SendExtractKind = {
  passport_extract: 'passport_extract',
  risk_notice: 'risk_notice',
  claim_notice: 'claim_notice',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type GenealogyNodeKind = typeof GenealogyNodeKind[keyof typeof GenealogyNodeKind];


export const GenealogyNodeKind = {
  item: 'item',
  lot: 'lot',
  partner_extract: 'partner_extract',
} as const;

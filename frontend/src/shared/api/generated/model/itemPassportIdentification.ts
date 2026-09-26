/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Идентификация (AD-16): под сомнением — изоляция до повторной идентификации.
 */
export type ItemPassportIdentification = typeof ItemPassportIdentification[keyof typeof ItemPassportIdentification];


export const ItemPassportIdentification = {
  unique: 'unique',
  probable: 'probable',
  ambiguous: 'ambiguous',
  unidentified: 'unidentified',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type IdentificationQuestionCause = typeof IdentificationQuestionCause[keyof typeof IdentificationQuestionCause];


export const IdentificationQuestionCause = {
  carrier_unreadable: 'carrier_unreadable',
  carrier_mismatch: 'carrier_mismatch',
  ambiguous_binding: 'ambiguous_binding',
  carrier_missing: 'carrier_missing',
} as const;

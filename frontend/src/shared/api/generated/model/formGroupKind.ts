/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type FormGroupKind = typeof FormGroupKind[keyof typeof FormGroupKind];


export const FormGroupKind = {
  charge: 'charge',
  batch_operation: 'batch_operation',
  transport: 'transport',
  other: 'other',
} as const;

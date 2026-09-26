/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ItemGroupKind = typeof ItemGroupKind[keyof typeof ItemGroupKind];


export const ItemGroupKind = {
  charge: 'charge',
  batch_operation: 'batch_operation',
  transport: 'transport',
  other: 'other',
} as const;

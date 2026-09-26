/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ReceiveMovementDestinationKind = typeof ReceiveMovementDestinationKind[keyof typeof ReceiveMovementDestinationKind];


export const ReceiveMovementDestinationKind = {
  station: 'station',
  workshop: 'workshop',
  warehouse: 'warehouse',
  isolator: 'isolator',
  other: 'other',
} as const;

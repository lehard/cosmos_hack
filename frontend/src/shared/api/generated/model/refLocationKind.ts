/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type RefLocationKind = typeof RefLocationKind[keyof typeof RefLocationKind];


export const RefLocationKind = {
  building: 'building',
  workshop: 'workshop',
  line: 'line',
  station: 'station',
  workplace: 'workplace',
  warehouse: 'warehouse',
  isolator: 'isolator',
  storage: 'storage',
} as const;

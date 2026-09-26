/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type DefineLocationKind = typeof DefineLocationKind[keyof typeof DefineLocationKind];


export const DefineLocationKind = {
  building: 'building',
  workshop: 'workshop',
  line: 'line',
  station: 'station',
  workplace: 'workplace',
  warehouse: 'warehouse',
  isolator: 'isolator',
  storage: 'storage',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type CoveragePointMethod = typeof CoveragePointMethod[keyof typeof CoveragePointMethod];


export const CoveragePointMethod = {
  camera: 'camera',
  cmm: 'cmm',
  radiography: 'radiography',
  ultrasonic: 'ultrasonic',
  penetrant: 'penetrant',
  leak_test: 'leak_test',
  torque: 'torque',
  visual_human: 'visual_human',
  supplier_documents: 'supplier_documents',
  laboratory: 'laboratory',
  other: 'other',
} as const;

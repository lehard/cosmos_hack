/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Визуальный контроль или контроль действий оператора; по умолчанию visionqc.
 */
export type AdmitPassportAnalyzerKind = typeof AdmitPassportAnalyzerKind[keyof typeof AdmitPassportAnalyzerKind];


export const AdmitPassportAnalyzerKind = {
  visionqc: 'visionqc',
  operatorvision: 'operatorvision',
} as const;

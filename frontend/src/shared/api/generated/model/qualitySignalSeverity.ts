/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type QualitySignalSeverity = typeof QualitySignalSeverity[keyof typeof QualitySignalSeverity];


export const QualitySignalSeverity = {
  critical: 'critical',
  major: 'major',
  minor: 'minor',
  unknown: 'unknown',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type QualitySignalBasisKind = typeof QualitySignalBasisKind[keyof typeof QualitySignalBasisKind];


export const QualitySignalBasisKind = {
  inspection_result: 'inspection_result',
  equipment_deviation: 'equipment_deviation',
  check_skipped: 'check_skipped',
  damage_on_receipt: 'damage_on_receipt',
  leak: 'leak',
  special_process_violation: 'special_process_violation',
  operator_report: 'operator_report',
} as const;

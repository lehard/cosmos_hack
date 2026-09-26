/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Вид проверки — для кнопки «Запросить измерение».
 */
export type NextCheckMeasurementKind = typeof NextCheckMeasurementKind[keyof typeof NextCheckMeasurementKind];


export const NextCheckMeasurementKind = {
  control_sample: 'control_sample',
  radiography: 'radiography',
  camera_reshoot: 'camera_reshoot',
  equipment_log: 'equipment_log',
  sample_inspection: 'sample_inspection',
  document_check: 'document_check',
  explanation: 'explanation',
  other: 'other',
} as const;

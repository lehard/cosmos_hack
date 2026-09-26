/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Класс операции из x-ant-action (AD-27, AD-40): read — чтение, record — запись без последствий на осях, protective — защитное, permissive — разрешающее, irreversible — необратимое.
 */
export type ActionClass = typeof ActionClass[keyof typeof ActionClass];


export const ActionClass = {
  read: 'read',
  record: 'record',
  protective: 'protective',
  permissive: 'permissive',
  irreversible: 'irreversible',
} as const;

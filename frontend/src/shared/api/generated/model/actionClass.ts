/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */

/**
 * Класс операции из x-ant-action (AD-27, AD-40); read — чтение.
 */
export type ActionClass = typeof ActionClass[keyof typeof ActionClass];


export const ActionClass = {
  read: 'read',
  record: 'record',
  protective: 'protective',
  permissive: 'permissive',
  irreversible: 'irreversible',
} as const;

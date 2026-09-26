/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * exact — поле ответа контракта; draft — по смыслу, путь уточняет модуль-владелец; manual — видно на экране.
 */
export type BoardRowMapping = typeof BoardRowMapping[keyof typeof BoardRowMapping];


export const BoardRowMapping = {
  exact: 'exact',
  draft: 'draft',
  manual: 'manual',
} as const;

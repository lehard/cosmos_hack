/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Ожидает исполнения, исполняется, исполнено.
 */
export type NCHandoffStatus = typeof NCHandoffStatus[keyof typeof NCHandoffStatus];


export const NCHandoffStatus = {
  waiting: 'waiting',
  in_progress: 'in_progress',
  done: 'done',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Откуда начинать: beginning — с самого начала (по умолчанию); start_step — с точки старта сценария (start_step в simulation.scenario.list): история сразу «у катастрофы», шаги до точки считаются пройденными. Пока только заготовки (профиль fixtures).
 */
export type StartRunStart = typeof StartRunStart[keyof typeof StartRunStart];


export const StartRunStart = {
  beginning: 'beginning',
  start_step: 'start_step',
} as const;

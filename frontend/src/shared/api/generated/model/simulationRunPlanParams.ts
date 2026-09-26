/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type SimulationRunPlanParams = {
/**
 * Весь план (прошедшее — done); по умолчанию — от текущего места.
 */
all?: boolean;
/**
 * Сколько строк (по умолчанию 50).
 * @minimum 0
 * @maximum 1000
 */
limit?: number;
};

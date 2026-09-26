/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Состояние качества (quality).
 */
export type NCItemAxesQuality = typeof NCItemAxesQuality[keyof typeof NCItemAxesQuality];


export const NCItemAxesQuality = {
  not_inspected: 'not_inspected',
  conforming: 'conforming',
  accepted_with_concession: 'accepted_with_concession',
  unable_to_assess: 'unable_to_assess',
  signal: 'signal',
  nonconforming: 'nonconforming',
} as const;

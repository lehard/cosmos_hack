/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Рекомендуемый исход: решение на точке или исход пересмотра.
 */
export type NCRecommendationOutcome = typeof NCRecommendationOutcome[keyof typeof NCRecommendationOutcome];


export const NCRecommendationOutcome = {
  accept: 'accept',
  accept_with_concession: 'accept_with_concession',
  reject: 'reject',
  insufficient_data: 'insufficient_data',
  upheld: 'upheld',
  revoked: 'revoked',
} as const;

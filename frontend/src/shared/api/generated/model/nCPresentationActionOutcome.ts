/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * outcome команды nonconformity.presentation.review.
 */
export type NCPresentationActionOutcome = typeof NCPresentationActionOutcome[keyof typeof NCPresentationActionOutcome];


export const NCPresentationActionOutcome = {
  upheld: 'upheld',
  revoked: 'revoked',
} as const;

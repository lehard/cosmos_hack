/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * upheld — оставить в силе; revoked — отозвать приёмку.
 */
export type ReviewPresentationOutcome = typeof ReviewPresentationOutcome[keyof typeof ReviewPresentationOutcome];


export const ReviewPresentationOutcome = {
  upheld: 'upheld',
  revoked: 'revoked',
} as const;

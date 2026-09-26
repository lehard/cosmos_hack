/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type NCConclusionVersionOutcome = typeof NCConclusionVersionOutcome[keyof typeof NCConclusionVersionOutcome];


export const NCConclusionVersionOutcome = {
  pass_to_next: 'pass_to_next',
  manual_review: 'manual_review',
  isolate: 'isolate',
  question_to_technologist: 'question_to_technologist',
} as const;

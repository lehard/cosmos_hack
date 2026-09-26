/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Ветка причины: why_made — почему возник (по умолчанию), why_missed — почему не обнаружили раньше.
 */
export type ConcludeCauseBranch = typeof ConcludeCauseBranch[keyof typeof ConcludeCauseBranch];


export const ConcludeCauseBranch = {
  why_made: 'why_made',
  why_missed: 'why_missed',
} as const;

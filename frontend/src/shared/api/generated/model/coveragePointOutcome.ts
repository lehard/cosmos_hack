/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Действующий исход полученного результата; «оценка невозможна» — нужен повторный контроль (FR-36).
 */
export type CoveragePointOutcome = typeof CoveragePointOutcome[keyof typeof CoveragePointOutcome];


export const CoveragePointOutcome = {
  defect_indicated: 'defect_indicated',
  no_defect_indicated: 'no_defect_indicated',
  unable_to_assess: 'unable_to_assess',
} as const;

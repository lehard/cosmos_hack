/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Помогло / не помогло / помогло не с первого раза / ещё проверяется.
 */
export type MemoryEntryOutcome = typeof MemoryEntryOutcome[keyof typeof MemoryEntryOutcome];


export const MemoryEntryOutcome = {
  helped: 'helped',
  not_helped: 'not_helped',
  helped_after_retry: 'helped_after_retry',
  in_progress: 'in_progress',
} as const;

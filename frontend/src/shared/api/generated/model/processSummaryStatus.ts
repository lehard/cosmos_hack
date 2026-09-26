/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * active — есть действующая версия; draft — ни одна версия ещё не введена; retired — все введённые выведены.
 */
export type ProcessSummaryStatus = typeof ProcessSummaryStatus[keyof typeof ProcessSummaryStatus];


export const ProcessSummaryStatus = {
  active: 'active',
  draft: 'draft',
  retired: 'retired',
} as const;

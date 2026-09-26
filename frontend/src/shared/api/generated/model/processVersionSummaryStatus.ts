/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ProcessVersionSummaryStatus = typeof ProcessVersionSummaryStatus[keyof typeof ProcessVersionSummaryStatus];


export const ProcessVersionSummaryStatus = {
  draft: 'draft',
  on_approval: 'on_approval',
  active: 'active',
  retired: 'retired',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ProcessVersionStatus = typeof ProcessVersionStatus[keyof typeof ProcessVersionStatus];


export const ProcessVersionStatus = {
  draft: 'draft',
  on_approval: 'on_approval',
  active: 'active',
  retired: 'retired',
} as const;

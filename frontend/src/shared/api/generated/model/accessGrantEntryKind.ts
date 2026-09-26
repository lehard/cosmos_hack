/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Что выдано или отозвано.
 */
export type AccessGrantEntryKind = typeof AccessGrantEntryKind[keyof typeof AccessGrantEntryKind];


export const AccessGrantEntryKind = {
  role: 'role',
  authority: 'authority',
  stamp: 'stamp',
  qualification: 'qualification',
  account: 'account',
  audit: 'audit',
  sod_rule: 'sod_rule',
} as const;

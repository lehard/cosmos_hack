/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Сфера выдачи (AD-11): обычная, ОТК, производство, администраторы и аудит.
 */
export type GrantAssessmentDomain = typeof GrantAssessmentDomain[keyof typeof GrantAssessmentDomain];


export const GrantAssessmentDomain = {
  ordinary: 'ordinary',
  qc: 'qc',
  production: 'production',
  admin: 'admin',
} as const;

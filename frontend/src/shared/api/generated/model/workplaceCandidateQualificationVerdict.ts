/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Квалификация на дату смены: ok — действует, expiring — истекает в 30 дней, expired — истекла, missing — нет в области поста. Контролёру квалификация не нужна (ok) — нужен документ согласования.
 */
export type WorkplaceCandidateQualificationVerdict = typeof WorkplaceCandidateQualificationVerdict[keyof typeof WorkplaceCandidateQualificationVerdict];


export const WorkplaceCandidateQualificationVerdict = {
  ok: 'ok',
  expiring: 'expiring',
  expired: 'expired',
  missing: 'missing',
} as const;

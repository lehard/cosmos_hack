/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */

/**
 * ok — последний отчёт верификатора «цело»; stale — свежего отчёта нет дольше двух интервалов.
 */
export type IntegrityStatusStatus = typeof IntegrityStatusStatus[keyof typeof IntegrityStatusStatus];


export const IntegrityStatusStatus = {
  ok: 'ok',
  violated: 'violated',
  stale: 'stale',
  unknown: 'unknown',
} as const;

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */
import type { IntegrityStatusStatus } from './integrityStatusStatus';

export interface IntegrityStatus {
  /** ok — последний отчёт верификатора «цело»; stale — свежего отчёта нет дольше двух интервалов. */
  status: IntegrityStatusStatus;
  checked_at?: string;
  /** Отпечаток отчёта верификатора */
  report_ref?: string;
  /**
     * Интервал проверок верификатора
     * @minimum 1
     */
  interval_seconds: number;
  /** Всегда true — показывается «по данным сервера» */
  server_side: true;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { IntegrityStatusStatus } from './integrityStatusStatus';

export interface IntegrityStatus {
  /** Время отчёта верификатора. */
  checked_at?: string;
  /**
     * Интервал проверок верификатора.
     * @minimum 1
     */
  interval_seconds: number;
  /** Отпечаток отчёта верификатора. */
  report_ref?: string;
  /** Всегда true — показывается «по данным сервера». */
  server_side: boolean;
  /** ok — последний отчёт верификатора «цело»; stale — свежего отчёта нет дольше двух интервалов. */
  status: IntegrityStatusStatus;
}

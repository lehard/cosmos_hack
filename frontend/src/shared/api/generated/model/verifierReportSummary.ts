/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { VerifierReportSummaryVerdict } from './verifierReportSummaryVerdict';

export interface VerifierReportSummary {
  checked_at: string;
  /** @minimum 0 */
  checked_up_to_seq: number;
  /** Главная находка словами: где нарушение (для строки списка). */
  headline?: string;
  /** Отпечаток подписанного отчёта. */
  report_digest: string;
  /** Всегда true: получено ant у хранителя — «по данным сервера». */
  server_side: boolean;
  /** «цело» только при нуле «не проверяемо», иначе «цело с оговорками». */
  verdict: VerifierReportSummaryVerdict;
  /** Хеш бинарника верификатора. */
  verifier_build?: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { IngestOutcome } from './ingestOutcome';

export interface ImportResult {
  /** Отпечаток файла streebog256:… */
  file_digest: string;
  /** Итог по строкам (index — номер строки). */
  rows: IngestOutcome[];
  /** @minimum 0 */
  rows_accepted: number;
  /** @minimum 0 */
  rows_duplicate: number;
  /** @minimum 0 */
  rows_rejected: number;
  /** @minimum 0 */
  rows_total: number;
  source_id: string;
}

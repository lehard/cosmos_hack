/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { QuarantineEntryState } from './quarantineEntryState';

export interface QuarantineEntry {
  /** seq, на котором построен ответ (для basis_seq команды переобработки, AD-39). */
  basis_seq: number;
  /** Исходное содержимое (только в чтении одной записи и при правах). */
  content?: string;
  detail?: string;
  event_id?: string;
  event_type?: string;
  /** Отпечаток канонического payload. */
  fingerprint: string;
  /** Адрес содержимого в хранилище материалов. */
  material_address: string;
  /** Код из contracts/errors.yaml. */
  problem_code: string;
  quarantine_id: string;
  quarantined_at: string;
  source_id: string;
  source_seq?: number;
  state: QuarantineEntryState;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { JournalRecordRef } from './journalRecordRef';

export interface NarrowOption {
  /** Основание — записи журнала (evidence_event_ids команды), с текстом. */
  evidence: JournalRecordRef[];
  /** Изделия, которые будут исключены. */
  item_ids: string[];
  /** Что сделать словами. */
  label: string;
  /** Основание словами — reason.text команды. */
  reason_text: string;
}

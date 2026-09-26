/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { JournalEntryViewChain } from './journalEntryViewChain';
import type { JournalEntryViewData } from './journalEntryViewData';
import type { JournalEntryViewEntryKind } from './journalEntryViewEntryKind';
import type { JournalEntryViewSignatureStatus } from './journalEntryViewSignatureStatus';

export interface JournalEntryView {
  basis_seq?: number;
  /** Критическое действие CA-‹n› (AD-28). */
  ca_ref?: string;
  /** @nullable */
  causation_id: string | null;
  chain: JournalEntryViewChain;
  committed_at: string;
  /** Исправляемая запись (FR-122). */
  corrects?: string;
  correlation_id: string;
  /** Содержимое data; нет — содержимое недоступно (нет KEK или прав). */
  data?: JournalEntryViewData;
  entry_kind: JournalEntryViewEntryKind;
  event_id: string;
  /** Тип записи каталога. */
  event_type: string;
  item_id?: string;
  occurred_at: string;
  /** Класс происхождения подписи (AD-2). */
  provenance_class: string;
  received_at: string;
  recorded_at: string;
  run_id?: string;
  /** @minimum 1 */
  schema_version: number;
  /** @minimum 1 */
  seq: number;
  /** Статус проверки подписи (FR-68); not_checked — профиль demo до эпика 05. */
  signature_status: JournalEntryViewSignatureStatus;
  /** key_id@версия подписантов. */
  signers: string[];
  source_id: string;
  /** Вид источника факта (FR-140). */
  source_kind?: string;
  stream: string;
}

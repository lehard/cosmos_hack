/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { JournalEntryView } from './journalEntryView';

export interface JournalEntryList {
  items: JournalEntryView[];
  next_cursor?: string;
}

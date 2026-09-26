/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { JournalHeadClockMode } from './journalHeadClockMode';

export interface JournalHead {
  /**
     * Последний номер CA-‹n›.
     * @minimum 0
     */
  ca_seq: number;
  /** Режим часов журнала (AD-37). */
  clock_mode: JournalHeadClockMode;
  /**
     * Доменное время последней записи; null — журнал пуст.
     * @nullable
     */
  recorded_at: string | null;
  /** @minimum 0 */
  seq: number;
}

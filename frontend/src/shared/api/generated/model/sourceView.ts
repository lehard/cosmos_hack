/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { SourceViewState } from './sourceViewState';

export interface SourceView {
  /** seq, на котором построен ответ (для basis_seq команд над источником, AD-39). */
  basis_seq: number;
  /** Оценка расхождения часов источника (FR-33). */
  clock_skew_ms?: number;
  /**
     * Дыр в source_seq (номер нет ни в журнале, ни в карантине).
     * @minimum 0
     */
  gap_count: number;
  /** Ключ устройства key_id@версия (акт ввода). */
  key_ref?: string;
  /** @nullable */
  last_received_at: string | null;
  /**
     * Последний source_seq.
     * @nullable
     */
  last_seq: number | null;
  /** @minimum 0 */
  quarantined: number;
  source_id: string;
  /** Вид источника (ручной ввод, станок, датчик, камера, внешняя система, импорт). */
  source_kind: string;
  state: SourceViewState;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { HypothesisBranch } from './hypothesisBranch';
import type { HypothesisCategory } from './hypothesisCategory';
import type { HypothesisStatus } from './hypothesisStatus';
import type { JournalRecordRef } from './journalRecordRef';

export interface Hypothesis {
  branch?: HypothesisBranch;
  category: HypothesisCategory;
  /**
     * Уверенность вывода в базисных пунктах — не вероятность вины.
     * @minimum 0
     * @maximum 10000
     */
  confidence_bp?: number;
  contradicting: JournalRecordRef[];
  hypothesis_id: string;
  /** Что измерить, чтобы проверить гипотезу. */
  measurement_hint?: string;
  statement?: string;
  status: HypothesisStatus;
  supporting: JournalRecordRef[];
}

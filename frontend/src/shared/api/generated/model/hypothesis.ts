/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { HypothesisBranch } from './hypothesisBranch';
import type { HypothesisCategory } from './hypothesisCategory';
import type { HypothesisChange } from './hypothesisChange';
import type { HypothesisStatus } from './hypothesisStatus';
import type { JournalRecordRef } from './journalRecordRef';
import type { NextCheck } from './nextCheck';

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
  /** Что меняло уверенность: версии вывода и решения людей по возрастанию времени. */
  history: HypothesisChange[];
  hypothesis_id: string;
  /** Что измерить, чтобы проверить гипотезу (устарело: next_check). */
  measurement_hint?: string;
  /** Что проверить следующим: проверка, что она разблокирует, сколько изделий может исключить. */
  next_check?: NextCheck;
  statement?: string;
  status: HypothesisStatus;
  supporting: JournalRecordRef[];
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { CorrectiveActionView } from './correctiveActionView';
import type { MemoryEntry } from './memoryEntry';
import type { QualitySummary } from './qualitySummary';
import type { RecurringProblem } from './recurringProblem';

export interface CorrectiveActionList {
  /** Доменное «сейчас», на которое посчитаны флаги. */
  as_of: string;
  basis_seq: number;
  items: CorrectiveActionView[];
  memory: MemoryEntry[];
  recurring: RecurringProblem[];
  summary: QualitySummary;
}

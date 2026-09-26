/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { MemoryEntryOutcome } from './memoryEntryOutcome';

export interface MemoryEntry {
  action_id: string;
  action_type: string;
  cause_category?: string;
  cycles: number;
  direction: string;
  factor?: string;
  incident_id: string;
  /** Помогло / не помогло / помогло не с первого раза / ещё проверяется. */
  outcome: MemoryEntryOutcome;
  title: string;
}

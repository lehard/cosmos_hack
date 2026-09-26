/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ProcessDiffEntry } from './processDiffEntry';

export interface ProcessVersionDiff {
  /** С чем сравнивается (по умолчанию — действующая). */
  against_id: string;
  entries: ProcessDiffEntry[];
  version_id: string;
}

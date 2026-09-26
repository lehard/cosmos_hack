/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { IngestOutcome } from './ingestOutcome';

export interface IngestResult {
  /** @minimum 0 */
  accepted: number;
  /** @minimum 0 */
  duplicates: number;
  items: IngestOutcome[];
  /** @minimum 0 */
  quarantined: number;
  /** @minimum 0 */
  rejected: number;
}

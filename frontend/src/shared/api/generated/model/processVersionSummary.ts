/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ProcessVersionSummaryStatus } from './processVersionSummaryStatus';
import type { VersionQuorum } from './versionQuorum';

export interface ProcessVersionSummary {
  created_at: string;
  /** @nullable */
  effective_from?: string | null;
  hash?: string;
  /** @minimum 0 */
  items_in_work: number;
  label: string;
  quorum?: VersionQuorum;
  status: ProcessVersionSummaryStatus;
  version_id: string;
}

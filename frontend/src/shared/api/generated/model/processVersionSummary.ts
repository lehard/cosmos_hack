/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ProcessVersionSummaryStatus } from './processVersionSummaryStatus';
import type { VersionQuorum } from './versionQuorum';

export interface ProcessVersionSummary {
  /** Лист утверждения (FR-23). */
  approval_document_id?: string;
  /** Версия, от которой начат черновик. */
  base_version_id?: string;
  created_at: string;
  /** @nullable */
  effective_from?: string | null;
  hash?: string;
  /** @minimum 0 */
  items_in_work: number;
  label: string;
  /** Процесс версии (UI-11). */
  process_id?: string;
  quorum?: VersionQuorum;
  status: ProcessVersionSummaryStatus;
  version_id: string;
}

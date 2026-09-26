/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ProcessElement } from './processElement';
import type { ProcessVersionStatus } from './processVersionStatus';
import type { VersionQuorum } from './versionQuorum';

export interface ProcessVersion {
  /** @nullable */
  author?: string | null;
  basis_seq: number;
  created_at: string;
  /** @nullable */
  effective_from?: string | null;
  /** Элементы в порядке маршрута. */
  elements: ProcessElement[];
  /** Хеш версии — H(байты XML как загружены), AD-17. */
  hash?: string;
  label: string;
  quorum?: VersionQuorum;
  status: ProcessVersionStatus;
  version_id: string;
}

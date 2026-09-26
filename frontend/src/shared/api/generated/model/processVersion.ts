/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ProcessElement } from './processElement';
import type { ProcessVersionStatus } from './processVersionStatus';
import type { VersionQuorum } from './versionQuorum';

export interface ProcessVersion {
  /** Лист утверждения (документ с маршрутом кворума, FR-23); есть с отправки на утверждение. */
  approval_document_id?: string;
  /** @nullable */
  author?: string | null;
  /** Версия, от которой начат черновик. */
  base_version_id?: string;
  basis_seq: number;
  created_at: string;
  /** @nullable */
  effective_from?: string | null;
  /** Элементы в порядке маршрута. */
  elements: ProcessElement[];
  /** Хеш версии — H(байты XML как загружены), AD-17. */
  hash?: string;
  label: string;
  /** Процесс версии (UI-11). */
  process_id?: string;
  quorum?: VersionQuorum;
  status: ProcessVersionStatus;
  version_id: string;
}

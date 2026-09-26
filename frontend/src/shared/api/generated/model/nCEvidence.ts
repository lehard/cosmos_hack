/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCRecordRef } from './nCRecordRef';
import type { NCRequirement } from './nCRequirement';
import type { NCSourceSignal } from './nCSourceSignal';

export interface NCEvidence {
  requirement?: NCRequirement;
  signals: NCSourceSignal[];
  /**
     * Похожих случаев (analysis.similar.list).
     * @minimum 0
     */
  similar_count: number;
  /** История этой зоны до и после операции. */
  zone_history: NCRecordRef[];
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DocumentRouteStageExternalParty } from './documentRouteStageExternalParty';
import type { DocumentRouteStageQuorum } from './documentRouteStageQuorum';
import type { DocumentStageSignature } from './documentStageSignature';

export interface DocumentRouteStage {
  attester_authority_id?: string;
  authority_id: string;
  /** Кто подписывает этап — для людей. */
  authority_label: string;
  /** Этап закрыт самим решением-источником. */
  by_source?: boolean;
  done: boolean;
  external_party?: DocumentRouteStageExternalParty;
  k?: number;
  paper_allowed: boolean;
  quorum: DocumentRouteStageQuorum;
  /**
     * Сколько засчитанных подписей нужно на этапе.
     * @minimum 1
     */
  required: number;
  role?: string;
  separation?: string[];
  /**
     * @minimum 0
     * @maximum 3
     */
  signature_level: number;
  signatures: DocumentStageSignature[];
  /** @minimum 1 */
  stage: number;
  stamp_kind?: string;
  title?: string;
}

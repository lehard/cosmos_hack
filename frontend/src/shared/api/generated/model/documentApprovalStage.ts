/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DocumentApprovalStageQuorum } from './documentApprovalStageQuorum';
import type { DocumentApprovalStageStatus } from './documentApprovalStageStatus';

export interface DocumentApprovalStage {
  /** Полномочие этапа. */
  authority: string;
  /** Кто может подписать этап (псевдонимы). */
  candidates: string[];
  /** Внешняя сторона: ВП, партнёр. */
  external_party?: string;
  /**
     * Уровень подписи (AD-13).
     * @minimum 0
     * @maximum 3
     */
  level: number;
  /** Бумага с заверением допустима на этапе (AD-43). */
  paper_allowed: boolean;
  /** Сколько подписей: 1 | все | k из n. */
  quorum: DocumentApprovalStageQuorum;
  /**
     * Сколько подписей нужно на этапе.
     * @minimum 1
     */
  required: number;
  /** Разделение обязанностей на этапе (FR-56). */
  separation_note?: string;
  /** Кто уже подписал (по текущему отпечатку). */
  signed_by: string[];
  /**
     * Номер этапа по порядку.
     * @minimum 1
     */
  stage: number;
  /** Вид цифрового клейма (для действий контроля, FR-145). */
  stamp_kind?: string;
  /** Состояние этапа. */
  status: DocumentApprovalStageStatus;
}

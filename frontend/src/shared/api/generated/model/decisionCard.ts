/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DocumentApprovalStage } from './documentApprovalStage';
import type { DocumentSummaryField } from './documentSummaryField';
import type { DrillRef } from './drillRef';

export interface DecisionCard {
  /** Что произойдёт после подписи. */
  after_signature: string;
  /** Основания — записи и объекты. */
  basis: DrillRef[];
  basis_seq: number;
  doc_digest: string;
  document_id: string;
  /** Срок решения. */
  due_at?: string;
  /** Кто ещё подписывает (подписант видит весь маршрут, AD-43). */
  other_signers: string[];
  /** Что решается — одной фразой. */
  question: string;
  /** Ваш этап маршрута. */
  stage: DocumentApprovalStage;
  subject: DrillRef;
  summary_fields: DocumentSummaryField[];
  title: string;
  /** @minimum 1 */
  version: number;
  /** Почему решение за вами: полномочие, этап маршрута. */
  why_you: string;
}

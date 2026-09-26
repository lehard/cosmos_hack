/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AwaitingAttestation } from './awaitingAttestation';
import type { DecisionEscalation } from './decisionEscalation';
import type { DecisionEvidence } from './decisionEvidence';
import type { DecisionProposal } from './decisionProposal';
import type { RoutedDocument } from './routedDocument';
import type { SimilarDecision } from './similarDecision';

export interface DecisionRequest {
  awaiting_attestation?: AwaitingAttestation;
  document: RoutedDocument;
  due_at?: string;
  escalation: DecisionEscalation;
  evidence: DecisionEvidence[];
  /** Ожидаемый подписант этапа — он же в QR бумажного экземпляра. */
  expected_signer?: string;
  /** Этап, который ждёт подписи текущего пользователя; нет — не ваш черёд. */
  my_stage?: number;
  proposal: DecisionProposal;
  similar_accepted: SimilarDecision[];
  similar_rejected: SimilarDecision[];
}

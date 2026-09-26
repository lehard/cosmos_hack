/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DecisionProposalKind } from './decisionProposalKind';

export interface DecisionProposal {
  /** Код предложения: вариант решения, concession, accept… */
  code: string;
  item_id: string;
  item_label: string;
  kind: DecisionProposalKind;
  nc_id?: string;
  nc_number?: string;
  summary: string;
}

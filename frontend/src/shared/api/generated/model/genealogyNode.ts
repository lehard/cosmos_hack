/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { GenealogyNodeKind } from './genealogyNodeKind';
import type { GenealogyNodeSummary } from './genealogyNodeSummary';

export interface GenealogyNode {
  kind: GenealogyNodeKind;
  label: string;
  /** Куда вошёл (сборка). */
  parent_ref?: string;
  /** Позиция в сборке. */
  position?: string;
  /** Для выписки партнёра — «происхождение подтверждено / не подтверждено» (AD-19). */
  provenance?: string;
  /** item_id или lot_id. */
  ref: string;
  summary?: GenealogyNodeSummary;
}

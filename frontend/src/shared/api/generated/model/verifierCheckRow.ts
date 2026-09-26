/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { VerifierCheckRowStatus } from './verifierCheckRowStatus';
import type { VerifierFinding } from './verifierFinding';

export interface VerifierCheckRow {
  ca_ref?: string;
  /** Проверка: цепочки, подписи, момент подписи, права подписанта, реакции, source_seq, документы, задержка записи, сборка, проекции. */
  check: string;
  /** @minimum 0 */
  count: number;
  details?: string;
  /** Находки проверки: где нарушение или оговорка (первые 20). */
  findings?: VerifierFinding[];
  /** цело / отвергнуто / не проверяемо. */
  status: VerifierCheckRowStatus;
}

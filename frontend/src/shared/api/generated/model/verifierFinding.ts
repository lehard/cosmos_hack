/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { VerifierFindingChain } from './verifierFindingChain';
import type { VerifierFindingStatus } from './verifierFindingStatus';

export interface VerifierFinding {
  /** Критическое действие CA-‹n›. */
  ca_ref?: string;
  /** Цепочка. */
  chain?: VerifierFindingChain;
  /** Код находки (chain_link_mismatch, checkpoint_mismatch.link, source_seq_gap.pending, …). */
  code: string;
  /** Что и где — по-русски. */
  detail: string;
  /** Запись журнала (event_id), если известна ant: переход к записи. */
  event_id?: string;
  /**
     * Номер записи в цепочке.
     * @minimum 0
     */
  seq?: number;
  /** rejected — нарушение, unverifiable — не проверяемо, note — пояснение при «цело». */
  status?: VerifierFindingStatus;
}

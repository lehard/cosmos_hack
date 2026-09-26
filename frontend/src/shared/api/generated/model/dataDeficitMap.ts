/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DeficitRow } from './deficitRow';

export interface DataDeficitMap {
  basis_seq: number;
  /** Всего расследований (разборов несоответствий). */
  investigations: number;
  rows: DeficitRow[];
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { BoardRow } from './boardRow';

export interface Board {
  basis_seq: number;
  /** @minimum 0 */
  failed: number;
  /** @minimum 0 */
  passed: number;
  /** @minimum 0 */
  pending: number;
  rows: BoardRow[];
  run_id: string;
}

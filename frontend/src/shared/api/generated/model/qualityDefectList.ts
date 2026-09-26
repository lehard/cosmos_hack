/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { QualityDefect } from './qualityDefect';

export interface QualityDefectList {
  /** @minimum 0 */
  defect_count: number;
  items: QualityDefect[];
  /** @minimum 0 */
  items_with_defect: number;
  next_cursor?: string;
}

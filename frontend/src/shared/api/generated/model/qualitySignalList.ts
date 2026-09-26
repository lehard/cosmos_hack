/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { QualitySignal } from './qualitySignal';

export interface QualitySignalList {
  items: QualitySignal[];
  next_cursor?: string;
}

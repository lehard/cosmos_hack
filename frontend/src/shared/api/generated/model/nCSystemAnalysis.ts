/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCConclusionVersion } from './nCConclusionVersion';

export interface NCSystemAnalysis {
  /** Альтернативные объяснения. */
  alternatives: string[];
  /** Нехватка сведений. */
  missing_information: string[];
  /** По возрастанию версии. */
  versions: NCConclusionVersion[];
  /** Почему система это предлагает — основания по-русски. */
  why: string[];
}

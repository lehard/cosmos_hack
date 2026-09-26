/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCConclusionVersion } from './nCConclusionVersion';
import type { NCSystemAnalysisMissingInformationItem } from './nCSystemAnalysisMissingInformationItem';

export interface NCSystemAnalysis {
  /** Альтернативные объяснения. */
  alternatives: string[];
  /** Нехватка сведений — перечисление (как missing_information в incident.hypothesis.computed). */
  missing_information: NCSystemAnalysisMissingInformationItem[];
  /** По возрастанию версии. */
  versions: NCConclusionVersion[];
  /** Почему система это предлагает — основания по-русски. */
  why: string[];
}

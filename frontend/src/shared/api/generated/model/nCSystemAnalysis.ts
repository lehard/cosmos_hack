/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCConclusionVersion } from './nCConclusionVersion';
import type { NCSystemAnalysisMissingInformationCodesItem } from './nCSystemAnalysisMissingInformationCodesItem';

export interface NCSystemAnalysis {
  /** Альтернативные объяснения. */
  alternatives: string[];
  /** Нехватка сведений — коды (те же, что missing_information_codes). */
  missing_information: string[];
  /** Нехватка сведений — перечисление (как missing_information в incident.hypothesis.computed). */
  missing_information_codes?: NCSystemAnalysisMissingInformationCodesItem[];
  /** По возрастанию версии. */
  versions: NCConclusionVersion[];
  /** Почему система это предлагает — основания по-русски. */
  why: string[];
}

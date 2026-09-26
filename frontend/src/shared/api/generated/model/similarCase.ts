/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { SimilarCaseCauseCategory } from './similarCaseCauseCategory';
import type { SimilarCaseResult } from './similarCaseResult';

export interface SimilarCase {
  /** @nullable */
  cause_category: SimilarCaseCauseCategory;
  cause_confirmed: boolean;
  /** @nullable */
  measure: string | null;
  nc_id: string;
  number: string;
  /** @nullable */
  result: SimilarCaseResult;
}

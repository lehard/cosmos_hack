/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { HypothesesMissingInformationItem } from './hypothesesMissingInformationItem';
import type { Hypothesis } from './hypothesis';
import type { SimilarCase } from './similarCase';

export interface Hypotheses {
  basis_seq: number;
  conclusion_is_categorical: boolean;
  hypotheses: Hypothesis[];
  missing_information: HypothesesMissingInformationItem[];
  nc_id: string;
  similar_cases: SimilarCase[];
  /** @minimum 0 */
  version: number;
}

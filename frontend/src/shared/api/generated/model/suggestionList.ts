/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { GeneratorInfo } from './generatorInfo';
import type { Suggestion } from './suggestion';

export interface SuggestionList {
  basis_seq: number;
  generators: GeneratorInfo[];
  items: Suggestion[];
}

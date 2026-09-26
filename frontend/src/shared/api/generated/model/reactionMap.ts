/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ReactionRule } from './reactionRule';

export interface ReactionMap {
  /** Ссылка на версию карты реакций. */
  ref: string;
  rules: ReactionRule[];
  version: string;
}

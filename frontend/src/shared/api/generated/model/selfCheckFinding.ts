/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { SelfCheckFindingCheck } from './selfCheckFindingCheck';
import type { SelfCheckFindingSeverity } from './selfCheckFindingSeverity';

export interface SelfCheckFinding {
  check: SelfCheckFindingCheck;
  severity: SelfCheckFindingSeverity;
  text: string;
}

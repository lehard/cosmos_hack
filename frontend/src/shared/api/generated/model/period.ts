/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { PeriodKind } from './periodKind';

export interface Period {
  from: string;
  kind: PeriodKind;
  to: string;
}

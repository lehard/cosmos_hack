/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { IntegrationCheckResult } from './integrationCheckResult';

export interface IntegrationCheck {
  at: string;
  detail?: string;
  endpoint?: string;
  result: IntegrationCheckResult;
  seq: number;
}

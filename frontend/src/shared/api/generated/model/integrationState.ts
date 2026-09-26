/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { IntegrationStateState } from './integrationStateState';
import type { IntegrationStateSystem } from './integrationStateSystem';

export interface IntegrationState {
  detail?: string;
  /** @nullable */
  since: string | null;
  /** disabled — система не включена (integrations.enabled) или канал выключен. */
  state: IntegrationStateState;
  system: IntegrationStateSystem;
}

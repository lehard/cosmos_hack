/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { FactorRef } from './factorRef';
import type { IncidentSummaryStatus } from './incidentSummaryStatus';

export interface IncidentSummary {
  common_factor?: FactorRef;
  incident_id: string;
  /** @minimum 0 */
  initial_size: number;
  label: string;
  opened_at: string;
  /** @minimum 0 */
  scope_version: number;
  /** @minimum 0 */
  size: number;
  status: IncidentSummaryStatus;
}

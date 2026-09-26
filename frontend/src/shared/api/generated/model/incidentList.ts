/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { IncidentSummary } from './incidentSummary';

export interface IncidentList {
  items: IncidentSummary[];
  next_cursor?: string;
}

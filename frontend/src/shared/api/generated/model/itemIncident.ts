/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ItemIncidentAction } from './itemIncidentAction';
import type { ItemIncidentStatus } from './itemIncidentStatus';

export interface ItemIncident {
  /** Что делать. */
  action: ItemIncidentAction;
  incident_id: string;
  /** @minimum 0 */
  scope_version: number;
  /** Что известно. */
  status: ItemIncidentStatus;
  /** Попало в область через компонент. */
  via_assembly_of?: string;
}

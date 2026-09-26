/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface UnboundEvent {
  /** carrier — по носителю; manual — человеком. */
  basis?: string;
  bound_to?: string;
  candidates: string[];
  carrier_ref?: string;
  event_id: string;
  event_type: string;
  occurred_at: string;
  run_id?: string;
  source_id?: string;
}

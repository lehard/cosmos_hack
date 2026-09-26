/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface ItemIntervention {
  closed_at?: string;
  intervention_id: string;
  opened_at: string;
  purpose?: string;
  retest_required?: boolean;
  zone_ids: string[];
}

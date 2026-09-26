/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface QualityDefect {
  defect_id: string;
  defect_type_code?: string;
  first_observation_event_id: string;
  identified_at: string;
  item_id: string;
  location?: string;
  /**
     * Наблюдений этого дефекта.
     * @minimum 1
     */
  observations: number;
  zone_id: string;
}

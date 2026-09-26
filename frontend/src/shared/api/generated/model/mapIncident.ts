/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface MapIncident {
  /** Основание последнего изменения области. */
  basis?: string;
  incident_id: string;
  label: string;
  /** @minimum 0 */
  scope_version: number;
  /**
     * Размер области сейчас (34 → 13 → 6).
     * @minimum 0
     */
  size: number;
  /** @minimum 0 */
  size_at_creation: number;
}

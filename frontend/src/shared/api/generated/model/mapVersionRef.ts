/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface MapVersionRef {
  is_current: boolean;
  /**
     * Изделий в работе по этой версии.
     * @minimum 0
     */
  items: number;
  label: string;
  process_version_id: string;
}

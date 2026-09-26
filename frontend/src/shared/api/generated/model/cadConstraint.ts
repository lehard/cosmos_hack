/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface CadConstraint {
  /** Зоны, к которым соединение закрывает доступ. */
  closes_zone_ids?: string[];
  constraint_id: string;
  first_item_type_id?: string;
  /** rework_limit, closes_access, install_order, fastening, torque. */
  kind: string;
  /** Лимит ремонтов зоны. */
  limit?: number;
  /** Файл сборки или шаг ТП нормативного слоя. */
  limit_source?: string;
  link_id: string;
  note?: string;
  quantity?: number;
  /** Шаг процесса, исполняющий ограничение. */
  step_key?: string;
  then_item_type_id?: string;
  tolerance_pct?: number;
  unit?: string;
  value?: number;
  zone_id?: string;
}

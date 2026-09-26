/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NcGroupInvestigation } from './ncGroupInvestigation';

export interface NcGroup {
  defect_type: string;
  /** Вид дефекта по-русски — классификатор видов дефектов. */
  defect_type_label?: string;
  equipment: string;
  /** Оборудование (или партия для входного брака) словами из справочника. */
  equipment_label?: string;
  group_key: string;
  /** Расследование (инцидент) группы; нет — группа без расследования. */
  incident_id?: string;
  investigation: NcGroupInvestigation;
  last_found_at: string;
  /** @minimum 0 */
  nc_count: number;
  /** Несоответствия группы — вход разбора обстоятельств и гипотез (эпик 12). */
  nc_ids: string[];
  operation: string;
  /** Имя шага BPMN действующей версии процесса. */
  operation_label?: string;
}

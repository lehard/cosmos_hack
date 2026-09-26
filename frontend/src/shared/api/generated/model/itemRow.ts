/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ItemStatus } from './itemStatus';

export interface ItemRow {
  item_id: string;
  item_type_id: string;
  /** Номер детали для людей. */
  label: string;
  lot_id?: string;
  /** Прогон сценария (AD-38). */
  run_id?: string;
  status: ItemStatus;
  /** Текущий шаг (AD-17). */
  step_key?: string;
  /** Версия процесса изделия (изделия прежних версий — с пометкой). */
  version_label?: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { MapItemIncidentStatus } from './mapItemIncidentStatus';
import type { MapItemPosition } from './mapItemPosition';
import type { MapItemSummary } from './mapItemSummary';

export interface MapItem {
  /** Статус в выбранном инциденте; нет — вне области (FR-9). */
  incident_status?: MapItemIncidentStatus;
  item_id: string;
  /** Номер детали для подписи точки. */
  label: string;
  position: MapItemPosition;
  /** Версия процесса, по которой изделие запущено (AD-17). */
  process_version_id: string;
  step_key: string;
  summary: MapItemSummary;
}

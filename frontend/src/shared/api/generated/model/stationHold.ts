/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { StationHoldLevel } from './stationHoldLevel';

export interface StationHold {
  /** Оборудование точки. */
  equipment_id?: string;
  hold_id: string;
  /** Инцидент, из-за которого остановка. */
  incident_id?: string;
  /** Стоп точки процесса или критическая остановка. */
  level: StationHoldLevel;
  /** Почему остановлено — словами. */
  reason: string;
  /** Условие снятия (что должно случиться). */
  release_condition?: string;
  /** Запись остановки в журнале (переход к записи). */
  set_event_id?: string;
  /** С какого момента остановлено. */
  since: string;
}

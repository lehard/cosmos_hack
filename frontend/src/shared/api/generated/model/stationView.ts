/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { StationAction } from './stationAction';
import type { StationCounters } from './stationCounters';
import type { StationHold } from './stationHold';
import type { StationSuggestion } from './stationSuggestion';

export interface StationView {
  /** Действия окна: только их показывает интерфейс. */
  actions: StationAction[];
  /** Действующие остановки точки процесса (FR-49). */
  active_holds: StationHold[];
  /** seq, на котором построен ответ. */
  basis_seq: number;
  /** Счётчики шага; нет — источник счётчиков недоступен (карточка узла process.node.read). */
  counters?: StationCounters;
  /** Оборудование шага, если известно. */
  equipment_id?: string;
  step_key: string;
  /** Имя шага по описанию процесса. */
  step_label?: string;
  /** Предложение системы: остановить или снять — решает человек. */
  suggestion?: StationSuggestion;
}

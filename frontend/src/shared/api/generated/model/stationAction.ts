/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { StationActionLevel } from './stationActionLevel';
import type { StationActionOperation } from './stationActionOperation';

export interface StationAction {
  /** Пройдёт гарды для вошедшего (остановка уже действует / не действует, полномочие снятия). */
  allowed: boolean;
  /** Что произойдёт: изделия, маршрут, точка чистоты, журнал. */
  consequences: string[];
  /** Оборудование — поле команды set. */
  equipment_id?: string;
  /** Остановка, которую снимает release (путь операции). */
  hold_id?: string;
  /** Надпись кнопки. */
  label: string;
  /** Уровень, с которым set остановит точку (поле команды). */
  level?: StationActionLevel;
  /** Операция API. */
  operation: StationActionOperation;
  /** Шаг — поле команды set. */
  step_key?: string;
  /** Почему доступно или почему нет — словами. */
  why_available: string;
}

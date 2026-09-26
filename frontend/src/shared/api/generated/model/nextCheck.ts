/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NextCheckMeasurementKind } from './nextCheckMeasurementKind';

export interface NextCheck {
  /**
     * Сколько изделий области проверка может исключить (оценка).
     * @minimum 0
     */
  could_exclude: number;
  /** Вид проверки — для кнопки «Запросить измерение». */
  measurement_kind: NextCheckMeasurementKind;
  /**
     * Размер текущей области риска.
     * @minimum 0
     */
  scope_size: number;
  /** Проверка словами. */
  text: string;
  /** Что проверка разблокирует: подтверждение причины, сужение области. */
  unlocks_text: string;
}

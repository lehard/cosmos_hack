/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface NCPresentationContext {
  /** Закрывающая точка (ЗТ). */
  closing_point: string;
  /** Запись предъявления (item.presentation.recorded). */
  event_id: string;
  /** Результаты методов контроля изделия (для method_event_ids решения). */
  method_event_ids?: string[];
  /**
     * Номер предъявления (повторное — больше 1).
     * @minimum 1
     */
  presentation_no: number;
  step_key: string;
}

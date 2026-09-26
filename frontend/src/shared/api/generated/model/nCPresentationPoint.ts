/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCPresentationPointAllowedResolutionsItem } from './nCPresentationPointAllowedResolutionsItem';

export interface NCPresentationPoint {
  /** Решения, которые пройдут гарды для вошедшего: разделение обязанностей, полномочие точки, блок, результаты методов, действующее разрешение на отклонение. */
  allowed_resolutions: NCPresentationPointAllowedResolutionsItem[];
  /** Закрывающая точка (ЗТ) — поле команды. */
  closing_point: string;
  /** Закрывающая точка для людей — имя узла процесса с этой ЗТ. */
  closing_point_label?: string;
  /** Запись предъявления (item.presentation.recorded). */
  event_id: string;
  /** Результаты методов контроля — method_event_ids команды. */
  method_event_ids: string[];
  /** Куда передаётся изделие при «Принять» — имя следующего шага процесса. */
  next_step_label?: string;
  /**
     * Номер предъявления (повторное — больше 1).
     * @minimum 1
     */
  presentation_no: number;
  step_key: string;
  /** Имя шага по описанию процесса. */
  step_label?: string;
}

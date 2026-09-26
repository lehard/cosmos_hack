/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface CausalWindow {
  /** causal_window_end — первая находка. */
  end: string;
  lower_bound_event_id?: string;
  /** causal_window_start — последнее подтверждённо нормальное состояние. */
  start: string;
  upper_bound_event_id?: string;
}

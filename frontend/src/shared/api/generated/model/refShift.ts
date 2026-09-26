/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface RefShift {
  ends_at: string;
  location_id: string;
  name?: string;
  /** Шаблон: повторять каждый день, пока начало не позже этого момента. */
  repeat_until?: string;
  shift_id: string;
  starts_at: string;
  /** Шаблон: только рабочие дни производственного календаря. */
  working_days_only?: boolean;
}

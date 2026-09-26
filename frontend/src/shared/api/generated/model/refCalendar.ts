/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { RefCalendarWeeklyDaysOffItem } from './refCalendarWeeklyDaysOffItem';

export interface RefCalendar {
  calendar_id: string;
  /** Даты YYYY-MM-DD. */
  non_working_days: string[];
  /** Даты YYYY-MM-DD. */
  shortened_days: string[];
  weekly_days_off: RefCalendarWeeklyDaysOffItem[];
  /** Рабочие дни на еженедельных выходных (перенос), YYYY-MM-DD. */
  working_days?: string[];
  year: number;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Для длительностей: смысл интервала — активная обработка / полное время на участке / иной интервал (пояснение — meaning_note).
 */
export type MetricValueMeaning = typeof MetricValueMeaning[keyof typeof MetricValueMeaning];


export const MetricValueMeaning = {
  active_processing: 'active_processing',
  time_at_station: 'time_at_station',
  other: 'other',
} as const;

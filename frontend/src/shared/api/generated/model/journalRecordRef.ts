/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { JournalRecordRefParams } from './journalRecordRefParams';

export interface JournalRecordRef {
  /** event_id записи. */
  event_id: string;
  /** Тип записи каталога, например equipment.deviation.detected. */
  event_type: string;
  /** Время возникновения (AD-37). */
  occurred_at: string;
  /** Параметры подписи: метод, параметр, значение, уставка, шаг… */
  params?: JournalRecordRefParams;
  /** Источник словами: «журнал «Сварочный источник ИС-2»», «камера КТ-3», псевдоним человека. */
  source_label?: string;
  /** Запись коротко словами для людей: что произошло, значение против уставки, источник. Нет — показать тип записи. */
  text?: string;
  /** Уточнение внутри типа: outcome контроля, deviation_kind отклонения, cycle_started / cycle_finished, condition. */
  variant?: string;
}

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
  /** Уточнение внутри типа: outcome контроля, deviation_kind отклонения, cycle_started / cycle_finished, condition. */
  variant?: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { CircumstanceRecordLane } from './circumstanceRecordLane';
import type { CircumstanceRecordParams } from './circumstanceRecordParams';

export interface CircumstanceRecord {
  /** Конец интервала (цикл, отклонение). */
  ended_at?: string;
  /** event_id записи. */
  event_id: string;
  /** Тип записи каталога, например equipment.deviation.detected. */
  event_type: string;
  /** Адреса материалов: кадры, протоколы. */
  evidence_refs?: string[];
  /** Позиция записи в журнале — для перехода к записи. */
  journal_seq?: number;
  /** Дорожка: изделие, человек, оборудование. */
  lane: CircumstanceRecordLane;
  /** Время возникновения (AD-37). */
  occurred_at: string;
  /** Параметры подписи: метод, параметр, значение, уставка, шаг… */
  params?: CircumstanceRecordParams;
  /** Связанные записи — подсвечиваются вместе с выбранной. */
  related_event_ids?: string[];
  /** Вид источника факта (FR-140). */
  source_kind?: string;
  /** Уточнение внутри типа: outcome контроля, deviation_kind отклонения, cycle_started / cycle_finished, condition. */
  variant?: string;
}

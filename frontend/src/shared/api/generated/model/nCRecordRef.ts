/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCParameterReading } from './nCParameterReading';
import type { NCRecordRefKind } from './nCRecordRefKind';
import type { NCRecordRefParams } from './nCRecordRefParams';
import type { NCRecordRefSourceKind } from './nCRecordRefSourceKind';

export interface NCRecordRef {
  /** Отметка «данных не было»: на момент решения записей этого источника не было; event_id — запись о потере связи источника или первая его запись, пришедшая позже. */
  absent?: boolean;
  /** Псевдоним автора решения или подписанта. */
  author?: string;
  event_id: string;
  /** Тип записи каталога. */
  event_type: string;
  /** Вид записи (AD-2): исходный факт, вывод системы, решение человека. */
  kind: NCRecordRefKind;
  occurred_at: string;
  params?: NCRecordRefParams;
  /** Параметр режима числами: уставка и наблюдённые значения (отклонение режима, сводка цикла). */
  reading?: NCParameterReading;
  /** Позиция в журнале — переход к записи. */
  seq?: number;
  /** Вид источника факта (FR-140). */
  source_kind?: NCRecordRefSourceKind;
  /** Краткое содержание для строки. */
  summary: string;
}

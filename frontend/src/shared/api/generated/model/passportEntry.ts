/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ItemSignature } from './itemSignature';
import type { PassportEntryKind } from './passportEntryKind';
import type { PassportEntryReliability } from './passportEntryReliability';
import type { PassportEntrySourceKind } from './passportEntrySourceKind';

export interface PassportEntry {
  /** Автор (псевдоним) или источник. */
  author?: string;
  /** Критическое действие (AD-28). */
  ca_ref?: string;
  /** Исправляемая запись (FR-122). */
  corrects?: string;
  event_id: string;
  /** Тип записи каталога. */
  event_type: string;
  /** Вид записи (AD-2): исходный сигнал, анализ системы, решение человека. */
  kind: PassportEntryKind;
  occurred_at: string;
  recorded_at: string;
  reliability?: PassportEntryReliability;
  seq: number;
  signatures: ItemSignature[];
  /** Пометка источника факта (FR-140). */
  source_kind?: PassportEntrySourceKind;
  step_key?: string;
  /** Краткое содержание для ленты истории. */
  summary: string;
}

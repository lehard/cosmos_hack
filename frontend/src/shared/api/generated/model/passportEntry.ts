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
  /** Как событие привязано к изделию: internal_id, carrier, post_context, time_window, manual (AD-41). */
  binding_basis?: string;
  /** Надёжность привязки: unique, probable, ambiguous, unidentified (FR-34). */
  binding_reliability?: string;
  /** Событие пришло без изделия и привязано позже (AD-41). */
  bound?: boolean;
  /** Критическое действие (AD-28). */
  ca_ref?: string;
  /** Кандидаты при неоднозначной привязке события (FR-34). */
  candidates?: string[];
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

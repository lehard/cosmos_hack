/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { JournalEventReading } from './journalEventReading';
import type { JournalEventViewKind } from './journalEventViewKind';
import type { JournalEventViewParams } from './journalEventViewParams';

export interface JournalEventView {
  /** Автор решения (псевдоним). */
  author?: string;
  /** event_id записи. */
  event_id: string;
  /** Тип записи словами (каталог типов). */
  event_title?: string;
  /** Тип записи каталога. */
  event_type: string;
  /** Адреса материалов: кадры, протоколы (материалы — materials.material.read). */
  evidence_refs: string[];
  /** Изделие записи. */
  item_id?: string;
  /**
     * Позиция записи в основной цепочке — переход к журналу (journal.entry.read).
     * @minimum 1
     */
  journal_seq: number;
  /** Вид записи (AD-2): исходный факт, вывод системы, решение человека, служебная. */
  kind: JournalEventViewKind;
  /** Запись пришла с опозданием (задержка записи). */
  late?: boolean;
  /** Время возникновения (AD-37). */
  occurred_at: string;
  /** Параметры записи: метод, параметр, значение, уставка, шаг… */
  params?: JournalEventViewParams;
  /** Параметр режима числами: уставка и наблюдённые значения. */
  reading?: JournalEventReading;
  /** Время записи в журнал; позже возникновения — запись опоздала. */
  recorded_at: string;
  /** key_id@версия подписантов. */
  signers: string[];
  /** Вид источника факта (FR-140). */
  source_kind?: string;
  /** Источник словами: журнал оборудования, камера, человек. */
  source_label?: string;
  /** Поток записи. */
  stream: string;
  /** Запись словами для людей: что произошло, значение против уставки, источник. */
  text?: string;
  /** Уточнение внутри типа: outcome контроля, вид отклонения… */
  variant?: string;
}

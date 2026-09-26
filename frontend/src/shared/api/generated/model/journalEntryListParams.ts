/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { Axis } from './axis';
import type { JournalEntryListEntryKind } from './journalEntryListEntryKind';
import type { JournalEntryListOrder } from './journalEntryListOrder';

export type JournalEntryListParams = {
/**
 * Изделие.
 * @maxLength 128
 */
item_id?: string;
/**
 * Поток: item:‹id›, incident:‹id›, global…
 * @maxLength 160
 */
stream?: string;
/**
 * Тип записи или префикс семейства (item., quality.).
 * @maxLength 128
 */
event_type?: string;
/**
 * Вид записи (AD-2).
 */
entry_kind?: JournalEntryListEntryKind;
/**
 * Записи после seq.
 * @minimum 0
 */
after_seq?: number;
/**
 * Одна запись по event_id: переход по causation_id, corrects, correlation_id (интерфейс 6).
 * @maxLength 64
 */
event_id?: string;
/**
 * asc (по умолчанию) — от старых, курсор — seq последней; desc — новые сверху, следующая страница — seq меньше курсора.
 */
order?: JournalEntryListOrder;
/**
 * Ось момента: occurred — «как было» (по умолчанию), recorded — «что мы знали» (AD-37).
 */
axis?: Axis;
/**
 * Момент (RFC 3339 UTC); пусто — «сейчас». Задан — воспроизведение: команды выключены (AD-21).
 */
as_of?: string;
/**
 * Прогон сценария: данные в его пространстве имён (AD-38).
 * @maxLength 128
 */
run_id?: string;
/**
 * Курсор следующей страницы из ответа; пусто — первая страница.
 * @maxLength 512
 */
cursor?: string;
/**
 * Размер страницы (по умолчанию 50).
 * @minimum 0
 * @maximum 500
 */
limit?: number;
};

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { Axis } from './axis';
import type { DocumentsDocumentListState } from './documentsDocumentListState';
import type { EntityKind } from './entityKind';

export type DocumentsDocumentListParams = {
/**
 * Вид объекта; пусто — все документы (реестр).
 */
subject?: EntityKind;
/**
 * Идентификатор объекта; пусто — все документы (реестр).
 * @maxLength 128
 */
id?: string;
/**
 * Реестр: документы изделия, включая документы его несоответствий.
 * @maxLength 128
 */
item_id?: string;
/**
 * Реестр: документы процесса.
 * @maxLength 128
 */
process_id?: string;
/**
 * Реестр: документы версии процесса.
 * @maxLength 128
 */
process_version_id?: string;
/**
 * Реестр: вид документа — id шаблона (nc-disposition) или template_ref.
 * @maxLength 128
 */
template?: string;
/**
 * Реестр: состояние документа.
 */
state?: DocumentsDocumentListState;
/**
 * Реестр: поиск по номеру, названию, объекту.
 * @maxLength 128
 */
q?: string;
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

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { Axis } from './axis';
import type { SigningKeyListStatus } from './signingKeyListStatus';
import type { SigningKeyListSubjectKind } from './signingKeyListSubjectKind';

export type SigningKeyListParams = {
/**
 * Вид субъекта.
 */
subject_kind?: SigningKeyListSubjectKind;
/**
 * Субъект.
 * @maxLength 128
 */
subject_id?: string;
/**
 * Статус.
 */
status?: SigningKeyListStatus;
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

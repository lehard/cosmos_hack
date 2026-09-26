/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { IngestQuarantineListState } from './ingestQuarantineListState';

export type IngestQuarantineListParams = {
/**
 * Источник.
 * @maxLength 128
 */
source_id?: string;
/**
 * Код причины.
 * @maxLength 128
 */
code?: string;
/**
 * Состояние.
 */
state?: IngestQuarantineListState;
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

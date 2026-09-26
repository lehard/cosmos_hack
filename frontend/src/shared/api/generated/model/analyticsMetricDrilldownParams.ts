/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AnalyticsMetricDrilldownPeriod } from './analyticsMetricDrilldownPeriod';
import type { Axis } from './axis';

export type AnalyticsMetricDrilldownParams = {
/**
 * Ключ среза; пусто — итог.
 * @maxLength 128
 */
slice?: string;
/**
 * Период: смена, сутки, неделя, месяц, произвольный (по умолчанию — смена).
 */
period?: AnalyticsMetricDrilldownPeriod;
/**
 * Начало произвольного периода (RFC 3339 UTC).
 */
from?: string;
/**
 * Конец произвольного периода.
 */
to?: string;
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

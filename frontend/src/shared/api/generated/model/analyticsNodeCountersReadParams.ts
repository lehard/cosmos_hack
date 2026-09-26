/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AnalyticsNodeCountersReadPeriod } from './analyticsNodeCountersReadPeriod';
import type { Axis } from './axis';

export type AnalyticsNodeCountersReadParams = {
/**
 * Версия процесса; пусто — действующая.
 * @maxLength 128
 */
process_version_id?: string;
/**
 * Период: смена, сутки, неделя, месяц, произвольный (по умолчанию — смена).
 */
period?: AnalyticsNodeCountersReadPeriod;
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
};

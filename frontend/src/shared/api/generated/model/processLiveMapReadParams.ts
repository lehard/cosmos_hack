/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { Axis } from './axis';
import type { ProcessLiveMapReadPeriod } from './processLiveMapReadPeriod';

export type ProcessLiveMapReadParams = {
/**
 * Период счётчиков (FR-3); по умолчанию shift.
 */
period?: ProcessLiveMapReadPeriod;
/**
 * Начало произвольного периода.
 */
from?: string;
/**
 * Конец произвольного периода.
 */
to?: string;
/**
 * Процесс (process.process.list, UI-11); не задан — основной процесс.
 * @maxLength 128
 */
process_id?: string;
/**
 * Версия процесса; не задана — действующая версия процесса.
 * @maxLength 128
 */
process_version_id?: string;
/**
 * Режим инцидента (FR-9).
 * @maxLength 128
 */
incident_id?: string;
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

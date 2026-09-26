/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { Axis } from './axis';

export type CrossitemTraceReadParams = {
/**
 * Партия.
 * @maxLength 128
 */
lot_id?: string;
/**
 * Плавка.
 * @maxLength 64
 */
heat_no?: string;
/**
 * Садка или иная временная группа.
 * @maxLength 128
 */
group_id?: string;
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

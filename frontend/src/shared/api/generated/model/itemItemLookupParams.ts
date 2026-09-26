/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */
import type { AsOfParameter } from './asOfParameter';
import type { AxisParameter } from './axisParameter';

export type ItemItemLookupParams = {
/**
 * Номер детали или содержимое DataMatrix
 * @minLength 1
 * @maxLength 256
 */
q: string;
/**
 * Ось момента (AD-37).
 */
axis?: AxisParameter;
/**
 * Момент (RFC 3339 UTC); пусто — «сейчас». Задан — воспроизведение, действия выключены.
 */
as_of?: AsOfParameter;
};

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */
import type { AsOfParameter } from './asOfParameter';
import type { AxisParameter } from './axisParameter';
import type { EntityKind } from './entityKind';

export type AccessPermissionListParams = {
/**
 * Вид объекта
 */
subject?: EntityKind;
/**
 * Идентификатор объекта
 * @maxLength 128
 */
id?: string;
/**
 * Ось момента (AD-37).
 */
axis?: AxisParameter;
/**
 * Момент (RFC 3339 UTC); пусто — «сейчас». Задан — воспроизведение, действия выключены.
 */
as_of?: AsOfParameter;
};

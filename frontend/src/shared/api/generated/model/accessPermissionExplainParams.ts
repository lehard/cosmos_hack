/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { EntityKind } from './entityKind';

export type AccessPermissionExplainParams = {
/**
 * x-ant-action id.
 * @maxLength 128
 */
action: string;
/**
 * Вид объекта.
 */
subject?: EntityKind;
/**
 * Идентификатор объекта.
 * @maxLength 128
 */
id?: string;
};

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { MaterialsMaterialUploadKind } from './materialsMaterialUploadKind';

export type MaterialsMaterialUploadParams = {
/**
 * Вид материала.
 */
kind: MaterialsMaterialUploadKind;
/**
 * Изделие.
 * @maxLength 128
 */
item_id?: string;
/**
 * Когда снято.
 */
captured_at?: string;
/**
 * Иллюстрация, а не доказательство.
 */
is_illustration?: boolean;
/**
 * Происхождение.
 * @maxLength 512
 */
provenance_note?: string;
};

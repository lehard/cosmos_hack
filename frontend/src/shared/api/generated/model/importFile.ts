/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ImportFileFormat } from './importFileFormat';

export interface ImportFile {
  /** Содержимое файла, base64; сохраняется в хранилище материалов по адресу H(байты). */
  content: string;
  /** Только проверить, не записывать. */
  dry_run?: boolean;
  /** @maxLength 256 */
  file_name: string;
  format: ImportFileFormat;
  /**
     * Шаблон сопоставления колонок с типом события.
     * @maxLength 128
     */
  mapping: string;
}

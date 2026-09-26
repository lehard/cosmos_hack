/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ItemDocumentRefStatus } from './itemDocumentRefStatus';

export interface ItemDocumentRef {
  /** Отпечаток документа (AD-12). */
  digest: string;
  document_id: string;
  status: ItemDocumentRefStatus;
  /** Шаблон@версия. */
  template: string;
  title: string;
}

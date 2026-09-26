/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { MaterialInfoKind } from './materialInfoKind';

export interface MaterialInfo {
  captured_at?: string;
  /** Иллюстрация, а не доказательство (NFR-UI-4). */
  is_illustration: boolean;
  item_id?: string;
  kind: MaterialInfoKind;
  /** Адрес содержимого streebog256:…; повтор загрузки тех же байтов — тот же адрес. */
  material_address: string;
  media_type: string;
  provenance_note?: string;
  /** @minimum 0 */
  size_bytes: number;
  /** Запись material.object.stored. */
  stored_event_id?: string;
}

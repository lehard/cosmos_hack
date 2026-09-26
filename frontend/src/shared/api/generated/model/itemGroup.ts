/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ItemGroupKind } from './itemGroupKind';

export interface ItemGroup {
  dissolved_at?: string;
  formed_at: string;
  group_id: string;
  item_ids: string[];
  kind: ItemGroupKind;
  /** Образец-свидетель. */
  witness_item_id?: string;
}

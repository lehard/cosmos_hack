/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface GenealogyHold {
  level: string;
  lot_id?: string;
  path?: string[];
  /** Основание снято у источника; блок снимает человек (AD-27). */
  released: boolean;
  /** lot — блок партии; component — блок компонента; split_parent — блок исходного изделия. */
  source: string;
  source_event_id: string;
  source_item_id?: string;
}

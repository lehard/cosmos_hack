/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface CadComponent {
  designation: string;
  /** Наш тип изделия по обозначению КД (FR-95). */
  item_type_id?: string;
  /** detail, subassembly, standard_part, fastener, purchased_equipment, other. */
  kind?: string;
  lot_tracked: boolean;
  /** make или buy. */
  make_or_buy?: string;
  material?: string;
  name?: string;
  /** Родитель в дереве состава. */
  parent_item_type_id?: string;
  position: string;
  /** @minimum 1 */
  quantity: number;
  shelf_life_tracked?: boolean;
  /** Количество на одну сборку верхнего уровня. */
  total_quantity?: number;
  /** serial, lot или none. */
  tracking?: string;
}

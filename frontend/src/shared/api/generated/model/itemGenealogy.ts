/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { GenealogyNode } from './genealogyNode';

export interface ItemGenealogy {
  /** Компоненты-экземпляры на всех уровнях (FR-45). */
  down?: string[];
  /** Временные группы (садки) изделия. */
  groups?: string[];
  item_id: string;
  /** Партии изделия и партионных компонентов. */
  lots?: string[];
  nodes: GenealogyNode[];
  /** Сборки, в которые вошло изделие, снизу вверх (FR-45). */
  up?: string[];
}

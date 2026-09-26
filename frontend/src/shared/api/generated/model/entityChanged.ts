/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { BackendMode } from './backendMode';
import type { EntityKind } from './entityKind';

export interface EntityChanged {
  /** Вид сущности — первый элемент ключа Vue Query. */
  entity: EntityKind;
  /**
     * Идентификатор сущности; для live_map и integrity — global.
     * @maxLength 128
     */
  id: string;
  /** Режим ведущих портов, отдавших изменение (AD-36). */
  mode?: BackendMode;
  /**
     * Прогон сценария, если изменение в его пространстве имён (AD-38).
     * @maxLength 128
     */
  run_id?: string;
  /**
     * Позиция журнала, после которой сущность изменилась; она же id события SSE.
     * @minimum 1
     */
  seq: number;
}

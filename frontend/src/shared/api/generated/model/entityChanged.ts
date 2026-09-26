/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */
import type { BackendMode } from './backendMode';
import type { EntityKind } from './entityKind';

/**
 * Сообщение SSE (contracts/events/common/sse-entity-changed.v1.json).
 */
export interface EntityChanged {
  entity: EntityKind;
  /** Идентификатор сущности; для live_map и integrity — global */
  id: string;
  /** @minimum 0 */
  seq: number;
  run_id?: string;
  mode?: BackendMode;
}

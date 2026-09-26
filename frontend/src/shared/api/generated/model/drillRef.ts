/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { EntityKind } from './entityKind';

export interface DrillRef {
  entity: EntityKind;
  /** @maxLength 128 */
  id: string;
}

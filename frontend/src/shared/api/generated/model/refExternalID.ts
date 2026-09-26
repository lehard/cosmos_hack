/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { RefExternalIDObjectKind } from './refExternalIDObjectKind';
import type { RefExternalIDSystem } from './refExternalIDSystem';

export interface RefExternalID {
  /** Конфликт соответствий — сигнал, не перезапись. */
  conflict: boolean;
  external_id: string;
  internal_id: string;
  mapped_at: string;
  object_kind: RefExternalIDObjectKind;
  system: RefExternalIDSystem;
}

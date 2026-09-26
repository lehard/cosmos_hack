/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ProcessDiffEntryKind } from './processDiffEntryKind';

export interface ProcessDiffEntry {
  /** thresholdChanged. */
  defectType?: string;
  /** elementAdded, elementRemoved, thresholdChanged, propertyChanged. */
  element?: string;
  /** Было (thresholdChanged, propertyChanged). */
  from?: unknown;
  kind: ProcessDiffEntryKind;
  /** propertyChanged: ключ process.properties.*. */
  property?: string;
  /** presentationPointAdded. */
  step?: string;
  /** Стало. */
  to?: unknown;
}

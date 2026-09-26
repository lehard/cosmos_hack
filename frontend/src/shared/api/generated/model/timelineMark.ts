/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DrillRef } from './drillRef';
import type { TimelineMarkKind } from './timelineMarkKind';

export interface TimelineMark {
  /** Момент события на выбранной оси. */
  at: string;
  kind: TimelineMarkKind;
  mark_id: string;
  /** Куда провалиться (FR-7). */
  ref?: DrillRef;
  /** Короткая подпись (узел, изделие). */
  title?: string;
}

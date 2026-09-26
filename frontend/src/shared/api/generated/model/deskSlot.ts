/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */
import type { Density } from './density';
import type { DeskArea } from './deskArea';
import type { DeskSlotSlice } from './deskSlotSlice';

export interface DeskSlot {
  /** @pattern ^[a-z][a-z0-9-]*$ */
  id: string;
  area: DeskArea;
  /**
     * id виджета из frontend/src/widgets/registry.ts
     * @pattern ^[a-z][a-z0-9-]*$
     */
  widget: string;
  density?: Density;
  /** Параметры среза — «одна правда, разные взгляды» */
  slice?: DeskSlotSlice;
}

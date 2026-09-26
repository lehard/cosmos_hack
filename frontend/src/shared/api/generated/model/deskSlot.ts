/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { Density } from './density';
import type { DeskArea } from './deskArea';
import type { DeskSlotSlice } from './deskSlotSlice';

export interface DeskSlot {
  area: DeskArea;
  density?: Density;
  /** @pattern ^[a-z][a-z0-9-]*$ */
  id: string;
  /** Параметры среза — «одна правда, разные взгляды». */
  slice?: DeskSlotSlice;
  /**
     * id виджета из frontend/src/widgets/registry.ts.
     * @pattern ^[a-z][a-z0-9-]*$
     */
  widget: string;
}

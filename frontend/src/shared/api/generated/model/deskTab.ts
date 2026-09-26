/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */
import type { Density } from './density';
import type { DeskLayout } from './deskLayout';
import type { DeskSlot } from './deskSlot';

export interface DeskTab {
  /** @pattern ^[a-z][a-z0-9-]*$ */
  id: string;
  /** Ключ текста интерфейса (ru.json) */
  title_key: string;
  layout: DeskLayout;
  density?: Density;
  slots: DeskSlot[];
}

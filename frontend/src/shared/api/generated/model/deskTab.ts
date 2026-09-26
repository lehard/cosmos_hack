/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { Density } from './density';
import type { DeskLayout } from './deskLayout';
import type { DeskSlot } from './deskSlot';

export interface DeskTab {
  density?: Density;
  /** @pattern ^[a-z][a-z0-9-]*$ */
  id: string;
  layout: DeskLayout;
  slots: DeskSlot[];
  /** Ключ текста интерфейса (ru.json). */
  title_key: string;
}

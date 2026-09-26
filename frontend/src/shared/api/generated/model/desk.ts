/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */
import type { Density } from './density';
import type { DeskTab } from './deskTab';
import type { DeskVersion } from './deskVersion';

/**
 * Стол роли — normative/desks/‹роль›.yaml (схема normative/desks/desk.schema.json).
 */
export interface Desk {
  version: DeskVersion;
  role: string;
  title_key: string;
  density: Density;
  /** Раздел встроенной справки роли */
  help_key?: string;
  /** @minItems 1 */
  tabs: DeskTab[];
}

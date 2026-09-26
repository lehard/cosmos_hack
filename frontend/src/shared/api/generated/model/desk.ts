/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { Density } from './density';
import type { DeskTab } from './deskTab';
import type { DeskVersion } from './deskVersion';

export interface Desk {
  density: Density;
  /** Раздел встроенной справки роли. */
  help_key?: string;
  /** id роли из политики. */
  role: string;
  /** @minItems 1 */
  tabs: DeskTab[];
  /** Ключ текста названия стола. */
  title_key: string;
  /** Версия формата стола. */
  version: DeskVersion;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ModuleMode } from './moduleMode';
import type { PortSetting } from './portSetting';
import type { SourceSwitch } from './sourceSwitch';

export interface SettingList {
  /** Включённые внешние системы (stand-ы или реальные адаптеры). */
  enabled_integrations: string[];
  modules: ModuleMode[];
  ports: PortSetting[];
  /** Источники, которые отключали или включали (ops.source.disabled / enabled). */
  sources?: SourceSwitch[];
}

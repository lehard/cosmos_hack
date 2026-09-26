/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { PortSettingZone } from './portSettingZone';

export interface PortSetting {
  /** Имя адаптера из deploy/config/ant.yaml. */
  adapter: string;
  /** Ключ порта: journal_store, work_feed, publisher, telemetry, access_control… */
  port: string;
  /** Замена (описано): Kafka, OTLP, PKCS#11, СКЗИ… */
  replace?: string;
  zone: PortSettingZone;
}

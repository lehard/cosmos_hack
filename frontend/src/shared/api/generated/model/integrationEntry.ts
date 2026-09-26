/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { IntegrationCheck } from './integrationCheck';
import type { IntegrationDecision } from './integrationDecision';
import type { IntegrationEntryChannel } from './integrationEntryChannel';
import type { IntegrationEntryState } from './integrationEntryState';
import type { IntegrationEntrySystem } from './integrationEntrySystem';
import type { IntegrationError } from './integrationError';

export interface IntegrationEntry {
  /**
     * basis_seq для следующей команды над интеграцией (AD-39).
     * @minimum 0
     */
  basis_seq: number;
  /** Живое состояние канала обмена (сверка ответной стороны, AD-18). */
  channel?: IntegrationEntryChannel;
  checked_at?: string;
  /** Решений не было — состояние по умолчанию профиля (prod — реальная, demo и fixtures — стенд). */
  default: boolean;
  detail?: string;
  /** Адрес ответной стороны без секретов. */
  endpoint?: string;
  /** Установлена конфигурацией (integrations.enabled); нет — «не установлена», кнопок нет. */
  installed: boolean;
  /** Последняя проверка соединения. */
  last_check?: IntegrationCheck;
  /** Последнее решение администратора. */
  last_decision?: IntegrationDecision;
  /** Последний сбой канала. */
  last_error?: IntegrationError;
  /** Последний обмен с системой. */
  last_exchange_at?: string;
  /**
     * Исходящих в карантине — ждут ручной переотправки (FR-96).
     * @minimum 0
     */
  quarantined?: number;
  /**
     * Исходящих в очереди (у выключенной копятся до включения).
     * @minimum 0
     */
  queued?: number;
  /** Можно переключить на реальную систему: её адрес задан конфигурацией. */
  real_available: boolean;
  /** Можно переключить на стенд: стенд задан конфигурацией и профиль не prod. */
  stand_available: boolean;
  /** enabled — включена (реальная система), disabled — выключена, stand — стенд (эмулятор). */
  state: IntegrationEntryState;
  /** Внешняя система. */
  system: IntegrationEntrySystem;
}

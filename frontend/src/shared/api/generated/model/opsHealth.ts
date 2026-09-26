/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { BackendMode } from './backendMode';
import type { ComponentState } from './componentState';
import type { IntegrationState } from './integrationState';
import type { OpsHealthProfile } from './opsHealthProfile';
import type { QueueState } from './queueState';
import type { SelfCheckView } from './selfCheckView';
import type { VerifierReportRef } from './verifierReportRef';

export interface OpsHealth {
  components: ComponentState[];
  integrations: IntegrationState[];
  mode: BackendMode;
  profile: OpsHealthProfile;
  /** @minimum 0 */
  quarantine_open: number;
  queues: QueueState[];
  /** Самопроверка после старта этой копии api (FR-109). */
  selfcheck?: SelfCheckView;
  /**
     * Изделия «обработка остановлена» (AD-45).
     * @minimum 0
     */
  stopped_items: number;
  /** Нет — отчёта ещё не было. */
  verifier?: VerifierReportRef;
  version: string;
}

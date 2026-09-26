/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { QueueStateScope } from './queueStateScope';

export interface QueueState {
  /**
     * Отставание курсора от головы журнала.
     * @minimum 0
     */
  lag_seq: number;
  /** Партиция воркера, глобальный потребитель, outbox. */
  name: string;
  /** @minimum 0 */
  pending: number;
  /**
     * Очередь исходящих: сообщений в карантине (ждут решения человека, FR-96).
     * @minimum 0
     */
  quarantined?: number;
  scope: QueueStateScope;
}

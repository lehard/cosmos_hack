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
  scope: QueueStateScope;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DsseEnvelope } from './dsseEnvelope';

export interface IngestBatch {
  /**
     * Подписанные конверты в порядке source_seq.
     * @minItems 1
     * @maxItems 1000
     */
  envelopes: DsseEnvelope[];
  /** Время отправки пачки по часам источника: по нему приём оценивает расхождение часов источника (FR-33); задержка досылки буфера сдвигом часов не считается. */
  sent_at?: string;
  /**
     * Источник пачки (устройство, шлюз, терминал); в прогоне — ‹run_id›/‹источник› (AD-38).
     * @maxLength 128
     */
  source_id: string;
}

/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface IngestMetrics {
  /** @minimum 0 */
  accepted: number;
  /**
     * Полнота: доля ожидаемых source_seq, которые есть, б. п.
     * @minimum 0
     * @maximum 10000
     */
  completeness_bp: number;
  /** @minimum 0 */
  duplicates: number;
  /**
     * ant_event_to_sse_seconds, бюджет FR-2 ≤ 2 с.
     * @minimum 0
     */
  event_to_screen_p95_ms: number;
  /**
     * Приём → запись, мс.
     * @minimum 0
     */
  latency_p50_ms: number;
  /** @minimum 0 */
  latency_p95_ms: number;
  /** @minimum 0 */
  quarantine_open: number;
  /** @minimum 0 */
  quarantined: number;
  /** @minimum 0 */
  received: number;
  /** @minimum 0 */
  rejected: number;
  window_from: string;
  window_to: string;
}

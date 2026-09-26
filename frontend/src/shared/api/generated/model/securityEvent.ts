/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DrillRef } from './drillRef';
import type { SecurityEventSeverity } from './securityEventSeverity';

export interface SecurityEvent {
  /** Критическое действие, если событие его породило. */
  ca_ref?: string;
  event_id: string;
  /** Тип записи семейства security. */
  event_type: string;
  object?: DrillRef;
  occurred_at: string;
  /** @minimum 1 */
  seq: number;
  severity: SecurityEventSeverity;
  source_id?: string;
  /** Краткое описание по-русски. */
  summary: string;
}
